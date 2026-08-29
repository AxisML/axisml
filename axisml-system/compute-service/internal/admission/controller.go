// Package admission implements durable, priority-ordered MLRun admission.
// It owns only the Queued -> Creating transition; runtime submission remains
// the MLRun outbox reconciler's responsibility.
package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/metrics"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

const (
	ReasonInventoryUnavailable        = "InventoryUnavailable"
	ReasonQuotaUnavailable            = "QuotaUnavailable"
	ReasonQuotaExceeded               = "QuotaExceeded"
	ReasonNoMatchingNode              = "NoMatchingNode"
	ReasonInsufficientResources       = "InsufficientResources"
	admissionAdvisoryLock       int64 = 0x415849534d4c5155 // "AXISMLQU"
)

type quotaKey struct {
	tenant string
	pool   string
}

type quotaResult struct {
	max corev1.ResourceList
	err error
}

// Controller periodically admits every currently fit-capable run in priority
// DESC, created_at ASC, id ASC order. A blocked high-priority run does not stop
// backfill candidates later in the ordered work set.
type Controller struct {
	db        *gorm.DB
	inventory extensions.ResourceInventory
	quotas    extensions.QuotaResolver
	log       logr.Logger
	interval  time.Duration
}

func NewController(db *gorm.DB, inventory extensions.ResourceInventory, quotas extensions.QuotaResolver, log logr.Logger, interval time.Duration) *Controller {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Controller{db: db, inventory: inventory, quotas: quotas, log: log, interval: interval}
}

func (c *Controller) NeedLeaderElection() bool { return true }

func (c *Controller) Start(ctx context.Context) error {
	c.runOnce(ctx)
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			c.runOnce(ctx)
		}
	}
}

func (c *Controller) runOnce(ctx context.Context) {
	var depth int64
	if err := c.db.WithContext(ctx).Model(&store.MLRun{}).
		Where("phase = ? AND deleted_at IS NULL", "Queued").Count(&depth).Error; err != nil {
		c.log.Error(err, "count queued MLRuns")
		return
	}
	metrics.MLRunQueueDepth.Set(float64(depth))
	if depth == 0 {
		return
	}

	var queued []store.MLRun
	if err := c.db.WithContext(ctx).
		Where("phase = ? AND deleted_at IS NULL", "Queued").
		Order("priority DESC, created_at ASC, id ASC").
		Find(&queued).Error; err != nil {
		c.log.Error(err, "list queued MLRuns")
		return
	}
	snapshot, err := c.inventory.Snapshot(ctx)
	if err != nil {
		c.log.Error(err, "read resource inventory")
		c.setAllQueueReason(ctx, queued, ReasonInventoryUnavailable, err.Error())
		return
	}

	// Quota reads may hit an informer/API-backed provider. Resolve every key
	// before taking the database lock, then consume this immutable map inside
	// the admission transaction.
	quotaResults := map[quotaKey]quotaResult{}
	for i := range queued {
		key := quotaKey{tenant: queued[i].Namespace, pool: poolFromLabels(queued[i].Labels)}
		if _, ok := quotaResults[key]; ok {
			continue
		}
		max, qerr := c.quotas.ResolveQuota(ctx, key.tenant, key.pool)
		quotaResults[key] = quotaResult{max: max, err: qerr}
	}

	err = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", admissionAdvisoryLock).Error; err != nil {
			return err
		}
		return c.admitLocked(ctx, tx, snapshot, quotaResults)
	})
	if err != nil {
		c.log.Error(err, "admit queued MLRuns")
	}
}

func (c *Controller) admitLocked(ctx context.Context, tx *gorm.DB, snapshot extensions.ResourceSnapshot, quotaResults map[quotaKey]quotaResult) error {
	nodes := newNodeState(snapshot)
	active, usage, actualCounts, err := loadActive(tx.WithContext(ctx), snapshot.Allocations)
	if err != nil {
		return err
	}
	for _, workload := range active {
		if !workload.reserveMissing {
			continue
		}
		for _, role := range workload.roles {
			missing := role.replicas - actualCounts[workload.id][role.name]
			if missing <= 0 {
				continue
			}
			if ok, _ := nodes.place(role.requests, missing, workload.nodeSelector, workload.tolerations); !ok {
				// A durable admitted reservation must never disappear merely because
				// its runtime instances are not visible yet. Exhaust the snapshot so
				// no new run is admitted against capacity already promised to it.
				nodes.exhaust()
				break
			}
		}
	}

	var queued []store.MLRun
	if err := tx.Where("phase = ? AND deleted_at IS NULL", "Queued").
		Order("priority DESC, created_at ASC, id ASC").
		Find(&queued).Error; err != nil {
		return err
	}
	for i := range queued {
		run := &queued[i]
		workload, err := runWorkload(run)
		if err != nil {
			return fmt.Errorf("decode queued MLRun %s/%s: %w", run.Namespace, run.Name, err)
		}
		key := quotaKey{tenant: run.Namespace, pool: workload.pool}
		quota, ok := quotaResults[key]
		if !ok || quota.err != nil {
			msg := "tenant quota is not available"
			if ok && quota.err != nil {
				msg = quota.err.Error()
			}
			if err := setQueueReason(tx, run, ReasonQuotaUnavailable, msg); err != nil {
				return err
			}
			continue
		}
		requested := workload.totalRequests()
		if exceeds(requested, quota.max) {
			if err := setQueueReason(tx, run, ReasonQuotaExceeded, "run request exceeds the tenant pool quota"); err != nil {
				return err
			}
			continue
		}
		if exceeds(sumLists(usage[key], requested), quota.max) {
			if err := setQueueReason(tx, run, ReasonQuotaUnavailable, "tenant pool quota is currently in use"); err != nil {
				return err
			}
			continue
		}

		trial := nodes.clone()
		placed := true
		reason := ReasonInsufficientResources
		for _, role := range workload.roles {
			var why string
			if placed, why = trial.place(role.requests, role.replicas, workload.nodeSelector, workload.tolerations); !placed {
				reason = why
				break
			}
		}
		if !placed {
			message := "no node has enough currently available resources"
			if reason == ReasonNoMatchingNode {
				message = "no schedulable node matches the run's selector and tolerations"
			}
			if err := setQueueReason(tx, run, reason, message); err != nil {
				return err
			}
			continue
		}

		status := statusWithQueue(run.StatusJSON, "", "")
		result := tx.Model(&store.MLRun{}).
			Where("id = ? AND phase = ? AND deleted_at IS NULL", run.ID, "Queued").
			Updates(map[string]any{
				"phase":  "Creating",
				"status": datatypes.JSON(status),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			nodes = trial
			usage[key] = sumLists(usage[key], requested)
			metrics.ReconcilerActions.WithLabelValues("mlrun", "admission", "admitted").Inc()
			c.log.V(1).Info("admitted MLRun", "namespace", run.Namespace, "name", run.Name, "priority", run.Priority)
		}
	}
	return nil
}

func (c *Controller) setAllQueueReason(ctx context.Context, rows []store.MLRun, reason, message string) {
	for i := range rows {
		if err := setQueueReason(c.db.WithContext(ctx), &rows[i], reason, message); err != nil {
			c.log.Error(err, "set MLRun queue reason", "name", rows[i].Name)
		}
	}
}

func setQueueReason(db *gorm.DB, run *store.MLRun, reason, message string) error {
	next := statusWithQueue(run.StatusJSON, reason, message)
	if string(next) == string(run.StatusJSON) {
		return nil
	}
	return db.Model(&store.MLRun{}).
		Where("id = ? AND phase = ?", run.ID, "Queued").
		Update("status", datatypes.JSON(next)).Error
}

func statusWithQueue(raw []byte, reason, message string) []byte {
	var status server.MLRunStatus
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &status)
	}
	status.QueueReason = reason
	status.Message = message
	b, _ := json.Marshal(status)
	return b
}

type roleRequest struct {
	name     string
	replicas int
	requests corev1.ResourceList
}

type desiredWorkload struct {
	id             string
	tenant         string
	pool           string
	nodeSelector   map[string]string
	tolerations    []corev1.Toleration
	roles          []roleRequest
	reserveMissing bool
}

func (w desiredWorkload) totalRequests() corev1.ResourceList {
	total := corev1.ResourceList{}
	for _, role := range w.roles {
		for i := 0; i < role.replicas; i++ {
			total = sumLists(total, role.requests)
		}
	}
	return total
}

func runWorkload(row *store.MLRun) (desiredWorkload, error) {
	var spec mlrunv1alpha1.MLRunSpec
	if err := json.Unmarshal(row.Spec, &spec); err != nil {
		return desiredWorkload{}, err
	}
	w := desiredWorkload{
		id: row.ID.String(), tenant: row.Namespace, pool: poolFromLabels(row.Labels),
		nodeSelector: spec.Scheduling.NodeSelector, tolerations: spec.Scheduling.Tolerations,
		reserveMissing: row.Phase != "Deleting",
	}
	for _, role := range spec.Roles {
		if role.Replicas > 0 {
			w.roles = append(w.roles, roleRequest{name: role.Name, replicas: int(role.Replicas), requests: copyList(role.Template.Resources.Requests)})
		}
	}
	stableRoleOrder(w.roles)
	return w, nil
}

func serviceWorkload(row *store.MLService) (desiredWorkload, error) {
	var spec mlservicev1alpha1.MLServiceSpec
	if err := json.Unmarshal(row.Spec, &spec); err != nil {
		return desiredWorkload{}, err
	}
	w := desiredWorkload{
		id: row.ID.String(), tenant: row.Namespace, pool: poolFromLabels(row.Labels),
		nodeSelector: spec.Scheduling.NodeSelector, tolerations: spec.Scheduling.Tolerations,
		reserveMissing: row.Phase != "Deleting",
	}
	for _, role := range spec.Roles {
		if role.Replicas > 0 {
			w.roles = append(w.roles, roleRequest{name: role.Name, replicas: int(role.Replicas), requests: copyList(role.Template.Resources.Requests)})
		}
	}
	stableRoleOrder(w.roles)
	return w, nil
}

func loadActive(db *gorm.DB, allocations []extensions.ResourceAllocation) ([]desiredWorkload, map[quotaKey]corev1.ResourceList, map[string]map[string]int, error) {
	var runs []store.MLRun
	if err := db.Where("phase IN ?", []string{"Creating", "Pending", "Running", "Canceling", "Deleting"}).Find(&runs).Error; err != nil {
		return nil, nil, nil, err
	}
	var services []store.MLService
	if err := db.Where("phase IN ?", []string{"Creating", "Pending", "Ready", "Degraded", "Failed", "Deleting"}).Find(&services).Error; err != nil {
		return nil, nil, nil, err
	}
	workloads := make([]desiredWorkload, 0, len(runs)+len(services))
	usage := map[quotaKey]corev1.ResourceList{}
	for i := range runs {
		w, err := runWorkload(&runs[i])
		if err != nil {
			return nil, nil, nil, err
		}
		workloads = append(workloads, w)
		key := quotaKey{tenant: w.tenant, pool: w.pool}
		usage[key] = sumLists(usage[key], w.totalRequests())
	}
	for i := range services {
		w, err := serviceWorkload(&services[i])
		if err != nil {
			return nil, nil, nil, err
		}
		workloads = append(workloads, w)
		key := quotaKey{tenant: w.tenant, pool: w.pool}
		usage[key] = sumLists(usage[key], w.totalRequests())
	}
	actual := map[string]map[string]int{}
	for _, w := range workloads {
		actual[w.id] = map[string]int{}
	}
	for _, allocation := range allocations {
		if roles, ok := actual[allocation.WorkloadID]; ok {
			roles[allocation.Role]++
		}
	}
	return workloads, usage, actual, nil
}

func poolFromLabels(raw []byte) string {
	var labels map[string]string
	_ = json.Unmarshal(raw, &labels)
	return labels[mlrunv1alpha1.LabelResourcePool]
}

// nodeState is intentionally private and mutable; clone supplies the trial
// placement used by work-conserving admission.
type nodeState struct {
	nodes []nodeAvailable
}

type nodeAvailable struct {
	name      string
	labels    map[string]string
	taints    []corev1.Taint
	available corev1.ResourceList
}

func newNodeState(snapshot extensions.ResourceSnapshot) *nodeState {
	s := &nodeState{nodes: make([]nodeAvailable, 0, len(snapshot.Nodes))}
	for _, node := range snapshot.Nodes {
		s.nodes = append(s.nodes, nodeAvailable{name: node.Name, labels: node.Labels, taints: node.Taints, available: copyList(node.Allocatable)})
	}
	for _, allocation := range snapshot.Allocations {
		for i := range s.nodes {
			if s.nodes[i].name == allocation.NodeName {
				subtractList(s.nodes[i].available, allocation.Requests)
				break
			}
		}
	}
	return s
}

func (s *nodeState) clone() *nodeState {
	out := &nodeState{nodes: make([]nodeAvailable, len(s.nodes))}
	for i := range s.nodes {
		out.nodes[i] = s.nodes[i]
		out.nodes[i].available = copyList(s.nodes[i].available)
	}
	return out
}

func (s *nodeState) exhaust() {
	for i := range s.nodes {
		for name := range s.nodes[i].available {
			quantity := s.nodes[i].available[name]
			quantity.Set(0)
			s.nodes[i].available[name] = quantity
		}
	}
}

func (s *nodeState) place(requests corev1.ResourceList, replicas int, selector map[string]string, tolerations []corev1.Toleration) (bool, string) {
	for replica := 0; replica < replicas; replica++ {
		eligible := false
		best := -1
		for i := range s.nodes {
			if !matches(s.nodes[i], selector, tolerations) {
				continue
			}
			eligible = true
			if !fits(s.nodes[i].available, requests) {
				continue
			}
			if best == -1 || betterFit(s.nodes[i], s.nodes[best], requests) {
				best = i
			}
		}
		if best == -1 {
			if !eligible {
				return false, ReasonNoMatchingNode
			}
			return false, ReasonInsufficientResources
		}
		subtractList(s.nodes[best].available, requests)
	}
	return true, ""
}

// betterFit chooses the node with the least residual capacity in the resource
// dimensions requested by this replica. This deterministic best-fit packing
// keeps larger free regions available for later queue entries.
func betterFit(candidate, current nodeAvailable, requests corev1.ResourceList) bool {
	for _, name := range orderedResourceNames(requests, nil) {
		candidateRemaining := candidate.available[name]
		candidateRemaining.Sub(requests[name])
		currentRemaining := current.available[name]
		currentRemaining.Sub(requests[name])
		if cmp := candidateRemaining.Cmp(currentRemaining); cmp != 0 {
			return cmp < 0
		}
	}
	return candidate.name < current.name
}

func matches(node nodeAvailable, selector map[string]string, tolerations []corev1.Toleration) bool {
	for key, value := range selector {
		if node.labels[key] != value {
			return false
		}
	}
	for _, taint := range node.taints {
		if taint.Effect != corev1.TaintEffectNoSchedule && taint.Effect != corev1.TaintEffectNoExecute {
			continue
		}
		if !isTolerated(taint, tolerations) {
			return false
		}
	}
	return true
}

func isTolerated(taint corev1.Taint, tolerations []corev1.Toleration) bool {
	for _, tol := range tolerations {
		if tol.Effect != "" && tol.Effect != taint.Effect {
			continue
		}
		if tol.Operator == corev1.TolerationOpExists && (tol.Key == "" || tol.Key == taint.Key) {
			return true
		}
		if (tol.Operator == "" || tol.Operator == corev1.TolerationOpEqual) && tol.Key == taint.Key && tol.Value == taint.Value {
			return true
		}
	}
	return false
}

func fits(available, requested corev1.ResourceList) bool {
	for name, request := range requested {
		availableQuantity := available[name]
		if availableQuantity.Cmp(request) < 0 {
			return false
		}
	}
	return true
}

func exceeds(used, max corev1.ResourceList) bool {
	for name, quantity := range used {
		limit := max[name]
		if quantity.Cmp(limit) > 0 {
			return true
		}
	}
	return false
}

func sumLists(a, b corev1.ResourceList) corev1.ResourceList {
	out := copyList(a)
	for name, quantity := range b {
		current := out[name]
		current.Add(quantity)
		out[name] = current
	}
	return out
}

func subtractList(dst corev1.ResourceList, used corev1.ResourceList) {
	for name, quantity := range used {
		current := dst[name]
		current.Sub(quantity)
		if current.Sign() < 0 {
			current.Set(0)
		}
		dst[name] = current
	}
}

func copyList(in corev1.ResourceList) corev1.ResourceList {
	out := make(corev1.ResourceList, len(in))
	for name, quantity := range in {
		out[name] = quantity.DeepCopy()
	}
	return out
}

// stableRoleOrder makes packing deterministic and puts GPU-, memory-, then
// CPU-heavy roles first. Other extended resources follow in name order.
func stableRoleOrder(roles []roleRequest) {
	sort.SliceStable(roles, func(i, j int) bool {
		for _, name := range orderedResourceNames(roles[i].requests, roles[j].requests) {
			left := roles[i].requests[name]
			right := roles[j].requests[name]
			if cmp := left.Cmp(right); cmp != 0 {
				return cmp > 0
			}
		}
		return roles[i].name < roles[j].name
	})
}

func orderedResourceNames(a, b corev1.ResourceList) []corev1.ResourceName {
	seen := map[corev1.ResourceName]bool{}
	for name := range a {
		seen[name] = true
	}
	for name := range b {
		seen[name] = true
	}
	preferred := []corev1.ResourceName{"nvidia.com/gpu", corev1.ResourceMemory, corev1.ResourceCPU}
	out := make([]corev1.ResourceName, 0, len(seen))
	for _, name := range preferred {
		if seen[name] {
			out = append(out, name)
			delete(seen, name)
		}
	}
	remaining := make([]string, 0, len(seen))
	for name := range seen {
		remaining = append(remaining, string(name))
	}
	sort.Strings(remaining)
	for _, name := range remaining {
		out = append(out, corev1.ResourceName(name))
	}
	return out
}
