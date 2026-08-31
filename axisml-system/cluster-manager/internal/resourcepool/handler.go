// Package resourcepool hosts the REST handlers for the ResourcePool CRD and
// its embedded `spec.units[]` array. All handlers are stateless and
// translate HTTP requests into K8s API calls on axisml.io/v1alpha1.ResourcePool.
package resourcepool

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	"github.com/axisml/axisml/axisml-system/cluster-manager/internal/promql"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// MetricsQuerier runs (tenant, pool) resource-utilisation queries against a
// metrics backend. Implemented by internal/promql.Querier; may be nil (standalone /
// unconfigured), in which case the metrics route reports metrics-unavailable.
type MetricsQuerier interface {
	Enabled() bool
	Series(ctx context.Context, namespace string, nodeSelector map[string]string, metric, rangeStr, stepStr string) (srv.PoolMetricSeries, error)
}

// Handler implements the /api/v1/resourcepools[/{pool}[/units...]]
// HTTP surface. It owns no state; all reads/writes go through the injected
// ResourcePoolProvider (Kubernetes CRD or static config). usage/metrics also read
// the Tenant CR (tenants) and query Prometheus (metrics).
type Handler struct {
	pools             extensions.ResourcePoolProvider
	tenants           extensions.TenantProvider
	metrics           MetricsQuerier
	validateResources extensions.ResourceListValidator
}

// NewHandler builds a resourcepool handler over the given stores. tenants powers
// the per-tenant usage endpoint; metrics (optional) powers the metrics endpoint.
func NewHandler(pools extensions.ResourcePoolProvider, tenants extensions.TenantProvider, metrics MetricsQuerier, validateResources extensions.ResourceListValidator) *Handler {
	return &Handler{pools: pools, tenants: tenants, metrics: metrics, validateResources: validateResources}
}

// Register attaches all routes to the provided /api/v1 group.
func (h *Handler) Register(rg *gin.RouterGroup) {
	pools := rg.Group("/resourcepools")
	pools.POST("", h.Create)
	pools.GET("", h.List)
	pools.GET("/:pool", h.Get)
	pools.PATCH("/:pool", h.Patch)
	pools.DELETE("/:pool", h.Delete)
	pools.GET("/:pool/usage", h.Usage)
	pools.GET("/:pool/metrics", h.Metrics)

	units := pools.Group("/:pool/units")
	units.POST("", h.CreateUnit)
	units.GET("", h.ListUnits)
	units.GET("/:unit", h.GetUnit)
	units.PATCH("/:unit", h.PatchUnit)
	units.DELETE("/:unit", h.DeleteUnit)
}

// Usage handles GET /api/v1/resourcepools/{pool}/usage?tenant=<id>. It returns
// the tenant's used-vs-total resource utilisation in the pool (ElasticQuota
// status Used against the folded quota ceiling).
func (h *Handler) Usage(c *gin.Context) {
	pool := c.Param("pool")
	tenant := c.Query("tenant")
	if tenant == "" {
		srv.AbortWithProblem(c, http.StatusBadRequest, "MissingTenant", "tenant query parameter is required", "")
		return
	}
	if _, err := h.getPool(c.Request.Context(), pool); err != nil {
		writeK8sError(c, err, pool)
		return
	}
	t, err := h.tenants.Get(c.Request.Context(), tenant)
	if err != nil {
		writeK8sError(c, err, tenant)
		return
	}
	c.JSON(http.StatusOK, srv.TenantPoolUsage(t, pool))
}

// Metrics handles GET /api/v1/resourcepools/{pool}/metrics?tenant=<id>. It
// returns a (tenant, pool) resource-utilisation time series from Prometheus.
func (h *Handler) Metrics(c *gin.Context) {
	pool := c.Param("pool")
	tenant := c.Query("tenant")
	if tenant == "" {
		srv.AbortWithProblem(c, http.StatusBadRequest, "MissingTenant", "tenant query parameter is required", "")
		return
	}
	if h.metrics == nil || !h.metrics.Enabled() {
		srv.AbortWithProblem(c, http.StatusServiceUnavailable, "MetricsUnavailable", "cluster metrics backend is not configured", "")
		return
	}
	p, err := h.getPool(c.Request.Context(), pool)
	if err != nil {
		writeK8sError(c, err, pool)
		return
	}
	series, err := h.metrics.Series(c.Request.Context(), tenant, p.Spec.NodeSelector,
		c.Query("metric"), c.Query("range"), c.Query("step"))
	if err != nil {
		writeMetricsError(c, err)
		return
	}
	c.JSON(http.StatusOK, series)
}

func writeMetricsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, promql.ErrUnavailable):
		srv.AbortWithProblem(c, http.StatusServiceUnavailable, "MetricsUnavailable", "cluster metrics backend is not configured", "")
	case errors.Is(err, promql.ErrBadRequest):
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidMetricsRequest", err.Error(), "")
	default:
		srv.AbortWithProblem(c, http.StatusBadGateway, "MetricsQueryFailed", "metrics query failed", err.Error())
	}
}

// ─────────────────────────────────────────────────────────────── Pools

// Create handles POST /api/v1/resourcepools.
func (h *Handler) Create(c *gin.Context) {
	var req srv.CreateResourcePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := srv.ValidateDNS1123Name("name", req.Name); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidName", err.Error(), "")
		return
	}
	if dup, name := firstDuplicateUnit(req.Units); dup {
		srv.AbortWithProblem(c, http.StatusBadRequest, "DuplicateUnit",
			"unit names must be unique within a pool", "duplicate: "+name)
		return
	}
	if req.Capacity != nil {
		if err := h.validateResources("capacity", req.Capacity); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidResources", err.Error(), "")
			return
		}
	}
	for i, u := range req.Units {
		if err := srv.ValidateDNS1123Name("units["+u.Name+"].name", u.Name); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidUnitName", err.Error(), "")
			return
		}
		if err := h.validateUnitResources(fmt.Sprintf("units[%d]", i), u.Requests, u.Limits); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidResources", err.Error(), "")
			return
		}
	}

	pool := srv.APIToPool(req, c.GetHeader(srv.HeaderUser))
	if err := h.pools.Create(c.Request.Context(), pool); err != nil {
		writeK8sError(c, err, req.Name)
		return
	}
	c.JSON(http.StatusCreated, srv.PoolToAPI(pool))
}

// Get handles GET /api/v1/resourcepools/{pool}.
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("pool")
	pool, err := h.getPool(c.Request.Context(), name)
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.PoolToAPI(pool))
}

// List handles GET /api/v1/resourcepools. Supports ?labelSelector,
// ?limit (1-500, default 100), and ?continue (opaque cursor returned by
// a previous page). Mirrors apis/cluster-manager.yaml.
func (h *Handler) List(c *gin.Context) {
	opts := metav1.ListOptions{}
	if sel := c.Query("labelSelector"); sel != "" {
		if _, err := labelSelectorFrom(sel); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidSelector",
				"labelSelector parse error", err.Error())
			return
		}
		opts.LabelSelector = sel
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidLimit",
				"limit must be an integer in [1, 500]", v)
			return
		}
		opts.Limit = int64(n)
	}
	opts.Continue = c.Query("continue")

	pools, err := h.pools.List(c.Request.Context(), opts)
	if err != nil {
		writeK8sError(c, err, "")
		return
	}

	resp := srv.ResourcePoolList{Items: make([]srv.ResourcePool, 0, len(pools.Items))}
	for i := range pools.Items {
		resp.Items = append(resp.Items, srv.PoolToAPI(&pools.Items[i]))
	}
	resp.Count = len(resp.Items)
	resp.ContinueToken = pools.Continue
	c.JSON(http.StatusOK, resp)
}

// Patch handles PATCH /api/v1/resourcepools/{pool}.
func (h *Handler) Patch(c *gin.Context) {
	name := c.Param("pool")
	var req srv.PatchResourcePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if req.Capacity != nil {
		if err := h.validateResources("capacity", req.Capacity); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidResources", err.Error(), "")
			return
		}
	}

	user := c.GetHeader(srv.HeaderUser)
	var result *axismlv1alpha1.ResourcePool
	err := h.mutateWithRetry(c.Request.Context(), name, func(pool *axismlv1alpha1.ResourcePool) error {
		applyPoolPatch(pool, req, user)
		result = pool
		return nil
	})
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.PoolToAPI(result))
}

// applyPoolPatch mutates `pool` in place per the PATCH request. Kept as a
// closure body so mutateWithRetry can re-run it against a fresh read on a
// 409 — replaying the *intent* of the patch rather than the snapshot of
// fields, which would silently overwrite any concurrent peer edit.
func applyPoolPatch(pool *axismlv1alpha1.ResourcePool, req srv.PatchResourcePoolRequest, user string) {
	if req.NodeSelector != nil {
		pool.Spec.NodeSelector = req.NodeSelector
	}
	if req.Capacity != nil {
		pool.Spec.Capacity = req.Capacity
	}
	if req.Labels != nil {
		pool.Labels = req.Labels
	}
	if pool.Annotations == nil {
		pool.Annotations = map[string]string{}
	}
	if req.Annotations != nil {
		// User-supplied annotations replace user-visible ones; preserve
		// reserved (description, last-modified-by) below.
		desc := pool.Annotations[srv.DescriptionAnnotation]
		lmb := pool.Annotations[srv.LastModifiedByAnnotation]
		pool.Annotations = map[string]string{}
		for k, v := range req.Annotations {
			pool.Annotations[k] = v
		}
		if desc != "" {
			pool.Annotations[srv.DescriptionAnnotation] = desc
		}
		if lmb != "" {
			pool.Annotations[srv.LastModifiedByAnnotation] = lmb
		}
	}
	if req.Description != nil {
		pool.Annotations[srv.DescriptionAnnotation] = *req.Description
	}
	if user != "" {
		pool.Annotations[srv.LastModifiedByAnnotation] = user
	}
}

// Delete handles DELETE /api/v1/resourcepools/{pool}.
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("pool")
	if err := h.pools.Delete(c.Request.Context(), name); err != nil {
		if apierrors.IsNotFound(err) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, name)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────── Units

// CreateUnit handles POST .../units. Optimistically patches the
// parent ResourcePool's spec.units[].
func (h *Handler) CreateUnit(c *gin.Context) {
	poolName := c.Param("pool")
	var req srv.CreateResourceUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := srv.ValidateDNS1123Name("name", req.Name); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidName", err.Error(), "")
		return
	}
	if err := h.validateUnitResources("unit", req.Requests, req.Limits); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidResources", err.Error(), "")
		return
	}

	var added axismlv1alpha1.ResourceUnit
	err := h.mutateWithRetry(c.Request.Context(), poolName, func(pool *axismlv1alpha1.ResourcePool) error {
		for _, u := range pool.Spec.Units {
			if u.Name == req.Name {
				// Surface as a non-retryable 409 by returning a sentinel
				// the outer handler maps via writeK8sError; here we use
				// apierrors.NewAlreadyExists so the predicate in
				// mutateWithRetry treats it as terminal (not conflict).
				return apierrors.NewAlreadyExists(
					axismlv1alpha1.GroupVersion.WithResource("resourcepools").GroupResource(),
					poolName+"/units/"+req.Name)
			}
		}
		pool.Spec.Units = append(pool.Spec.Units, srv.APIToUnit(req))
		if pool.Annotations == nil {
			pool.Annotations = map[string]string{}
		}
		if user := c.GetHeader(srv.HeaderUser); user != "" {
			pool.Annotations[srv.LastModifiedByAnnotation] = user
		}
		added = pool.Spec.Units[len(pool.Spec.Units)-1]
		return nil
	})
	if err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	c.JSON(http.StatusCreated, srv.UnitToAPI(added))
}

// ListUnits handles GET .../units. Returns pool.spec.units[].
func (h *Handler) ListUnits(c *gin.Context) {
	pool, err := h.getPool(c.Request.Context(), c.Param("pool"))
	if err != nil {
		writeK8sError(c, err, c.Param("pool"))
		return
	}
	items := make([]srv.ResourceUnit, 0, len(pool.Spec.Units))
	for _, u := range pool.Spec.Units {
		items = append(items, srv.UnitToAPI(u))
	}
	c.JSON(http.StatusOK, srv.ResourceUnitList{Items: items, Count: len(items)})
}

// GetUnit handles GET .../units/{unit}.
func (h *Handler) GetUnit(c *gin.Context) {
	pool, err := h.getPool(c.Request.Context(), c.Param("pool"))
	if err != nil {
		writeK8sError(c, err, c.Param("pool"))
		return
	}
	for _, u := range pool.Spec.Units {
		if u.Name == c.Param("unit") {
			c.JSON(http.StatusOK, srv.UnitToAPI(u))
			return
		}
	}
	srv.AbortWithProblem(c, http.StatusNotFound, "UnitNotFound",
		"unit not found in pool", c.Param("unit"))
}

// PatchUnit handles PATCH .../units/{unit}.
func (h *Handler) PatchUnit(c *gin.Context) {
	poolName, unitName := c.Param("pool"), c.Param("unit")
	var req srv.PatchResourceUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := h.validateUnitResources("unit", req.Requests, req.Limits); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidResources", err.Error(), "")
		return
	}

	user := c.GetHeader(srv.HeaderUser)
	var patched axismlv1alpha1.ResourceUnit
	err := h.mutateWithRetry(c.Request.Context(), poolName, func(pool *axismlv1alpha1.ResourcePool) error {
		idx := -1
		for i := range pool.Spec.Units {
			if pool.Spec.Units[i].Name == unitName {
				idx = i
				break
			}
		}
		if idx == -1 {
			return apierrors.NewNotFound(
				axismlv1alpha1.GroupVersion.WithResource("resourcepools").GroupResource(),
				poolName+"/units/"+unitName)
		}
		applyUnitPatch(&pool.Spec.Units[idx], req)
		if user != "" {
			if pool.Annotations == nil {
				pool.Annotations = map[string]string{}
			}
			pool.Annotations[srv.LastModifiedByAnnotation] = user
		}
		patched = pool.Spec.Units[idx]
		return nil
	})
	if err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	c.JSON(http.StatusOK, srv.UnitToAPI(patched))
}

// applyUnitPatch mutates one unit in place per the PATCH request. Kept
// as a closure body so mutateWithRetry can re-run it against a fresh
// read on 409 — replaying intent rather than overwriting a snapshot.
func applyUnitPatch(u *axismlv1alpha1.ResourceUnit, req srv.PatchResourceUnitRequest) {
	if req.Requests != nil {
		u.Requests = req.Requests
	}
	if req.Limits != nil {
		u.Limits = req.Limits
	}
	if req.NodeSelector != nil {
		u.NodeSelector = req.NodeSelector
	}
	if req.Annotations != nil {
		desc := ""
		if u.Annotations != nil {
			desc = u.Annotations[srv.DescriptionAnnotation]
		}
		u.Annotations = map[string]string{}
		for k, v := range req.Annotations {
			u.Annotations[k] = v
		}
		if desc != "" {
			u.Annotations[srv.DescriptionAnnotation] = desc
		}
	}
	if req.Description != nil {
		if u.Annotations == nil {
			u.Annotations = map[string]string{}
		}
		u.Annotations[srv.DescriptionAnnotation] = *req.Description
	}
}

// DeleteUnit handles DELETE .../units/{unit}.
func (h *Handler) DeleteUnit(c *gin.Context) {
	poolName, unitName := c.Param("pool"), c.Param("unit")
	user := c.GetHeader(srv.HeaderUser)
	err := h.mutateWithRetry(c.Request.Context(), poolName, func(pool *axismlv1alpha1.ResourcePool) error {
		idx := -1
		for i := range pool.Spec.Units {
			if pool.Spec.Units[i].Name == unitName {
				idx = i
				break
			}
		}
		if idx == -1 {
			// Idempotent: nothing to remove, treat the closure as a no-op
			// so the outer apierrors.NewNotFound becomes the public 204.
			return errNoMutationNeeded
		}
		pool.Spec.Units = append(pool.Spec.Units[:idx], pool.Spec.Units[idx+1:]...)
		if user != "" {
			if pool.Annotations == nil {
				pool.Annotations = map[string]string{}
			}
			pool.Annotations[srv.LastModifiedByAnnotation] = user
		}
		return nil
	})
	if err != nil {
		if apierrors.IsNotFound(err) || errors.Is(err, errNoMutationNeeded) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, poolName)
		return
	}
	c.Status(http.StatusNoContent)
}

// errNoMutationNeeded short-circuits mutateWithRetry's Patch call when
// the closure decides no change is required (e.g. idempotent DELETE).
var errNoMutationNeeded = errors.New("no mutation needed")

// ─────────────────────────────────────────────────────────────── helpers

func (h *Handler) getPool(ctx context.Context, name string) (*axismlv1alpha1.ResourcePool, error) {
	return h.pools.Get(ctx, name)
}

func (h *Handler) validateUnitResources(field string, requests, limits corev1.ResourceList) error {
	if h.validateResources == nil {
		return nil
	}
	if err := h.validateResources(field+".requests", requests); err != nil {
		return err
	}
	return h.validateResources(field+".limits", limits)
}

// mutateWithRetry runs `mutate` against a fresh ResourcePool read; if the
// Patch comes back with a 409 (resourceVersion conflict), it re-reads and
// reruns the mutation once. Design (cluster-manager.md §4.2) requires
// JSON Patch + resourceVersion optimistic lock + one retry to keep
// concurrent unit-edit UX usable.
//
// Re-running the closure (rather than re-applying a fixed Spec snapshot)
// preserves concurrent peer edits: if peer adds a unit between attempt 1
// and attempt 2, the closure sees that unit on the fresh read and only
// layers the caller's mutation on top.
//
// If `mutate` returns errNoMutationNeeded, the Patch call is skipped and
// nil is returned (idempotent no-op).
func (h *Handler) mutateWithRetry(ctx context.Context, poolName string, mutate func(p *axismlv1alpha1.ResourcePool) error) error {
	for attempt := 0; attempt < 2; attempt++ {
		pool, err := h.getPool(ctx, poolName)
		if err != nil {
			return err
		}
		base := pool.DeepCopy()
		if err := mutate(pool); err != nil {
			if errors.Is(err, errNoMutationNeeded) {
				return nil
			}
			return err
		}
		err = h.pools.Patch(ctx, pool, base)
		if err == nil {
			return nil
		}
		if !apierrors.IsConflict(err) {
			return err
		}
		// Conflict on the first attempt is recoverable; refresh and retry.
	}
	return apierrors.NewConflict(
		axismlv1alpha1.GroupVersion.WithResource("resourcepools").GroupResource(),
		poolName,
		errors.New("resourceVersion conflict after one retry"),
	)
}

func writeK8sError(c *gin.Context, err error, name string) {
	switch {
	case errors.Is(err, extensions.ErrCapabilityUnavailable):
		srv.AbortWithProblem(c, http.StatusConflict, "CapabilityUnavailable",
			"operation not supported in this deployment form", name)
	case apierrors.IsAlreadyExists(err):
		srv.AbortWithProblem(c, http.StatusConflict, "AlreadyExists",
			"resource already exists", name)
	case apierrors.IsNotFound(err):
		srv.AbortWithProblem(c, http.StatusNotFound, "NotFound",
			"resource not found", name)
	case apierrors.IsConflict(err):
		srv.AbortWithProblem(c, http.StatusConflict, "OptimisticLockConflict",
			"resource was modified concurrently — retry", name)
	case apierrors.IsInvalid(err):
		srv.AbortWithProblem(c, http.StatusUnprocessableEntity, "Invalid",
			"K8s API rejected the resource", err.Error())
	case apierrors.IsBadRequest(err):
		srv.AbortWithProblem(c, http.StatusBadRequest, "BadRequest", err.Error(), "")
	default:
		// Map K8s "no kind X in scheme" / generic errors to 500.
		srv.AbortWithProblem(c, http.StatusInternalServerError, "K8sError",
			"unexpected error from Kubernetes API", err.Error())
	}
}

func firstDuplicateUnit(units []srv.CreateResourceUnitRequest) (bool, string) {
	seen := map[string]struct{}{}
	for _, u := range units {
		if _, ok := seen[u.Name]; ok {
			return true, u.Name
		}
		seen[u.Name] = struct{}{}
	}
	return false, ""
}
