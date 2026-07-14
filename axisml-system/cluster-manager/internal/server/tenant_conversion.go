package server

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

// TenantToAPI renders a Tenant CR into its REST representation. Quotas are
// always anchored by spec.quotas[]; the round-trip annotation only chooses the
// units view for pools that were configured through unit selections.
func TenantToAPI(t *tenantv1alpha1.Tenant) Tenant {
	dto := Tenant{
		Name:            t.Name,
		Namespace:       t.Spec.Namespace,
		Quotas:          quotasToAPI(t.Spec.Quotas, t.Annotations),
		Labels:          t.Labels,
		Annotations:     stripReservedAnnotations(t.Annotations),
		ResourceVersion: t.ResourceVersion,
		Phase:           string(t.Status.Phase),
		CreatedAt:       t.CreationTimestamp.Time,
	}
	if !isZeroInitResources(t.Spec.InitResources) {
		ir := t.Spec.InitResources
		dto.InitResources = &ir
	}
	dto.Status = tenantStatusToAPI(t.Status)
	return dto
}

// APIToTenant builds a fresh Tenant CR from a create request. `folded` is the
// pre-computed ElasticQuota min/max (see FoldQuotas); `quotaAnno` is the
// JSON-encoded business-form selection stored for round-trip GET.
func APIToTenant(req CreateTenantRequest, folded []tenantv1alpha1.QuotaSpec, quotaAnno, lastModifiedBy string) *tenantv1alpha1.Tenant {
	ns := tenantv1alpha1.NamespaceSpec{Name: req.Name}
	if req.Namespace != nil {
		ns = *req.Namespace
		if ns.Name == "" {
			ns.Name = req.Name
		}
	}
	cr := &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      copyMap(req.Labels),
			Annotations: tenantAnnotations(req.Annotations, lastModifiedBy, quotaAnno),
		},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: ns,
			Quotas:    folded,
		},
	}
	if req.InitResources != nil {
		cr.Spec.InitResources = *req.InitResources
	}
	return cr
}

// FoldQuotas compiles quota input into the canonical Tenant CR shape. Unit
// selections are folded against ResourcePool.spec.units[]; direct quota inputs
// pass through after validation. The result is written 1:1 to
// Tenant.spec.quotas[] for tenant-operator to render.
func FoldQuotas(selections []Quota, pools map[string]*axismlv1alpha1.ResourcePool) ([]tenantv1alpha1.QuotaSpec, error) {
	out := make([]tenantv1alpha1.QuotaSpec, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))
	for _, q := range selections {
		if _, dup := seen[q.Pool]; dup {
			return nil, &QuotaError{Reason: QuotaDuplicatePool, Pool: q.Pool}
		}
		seen[q.Pool] = struct{}{}

		pool, ok := pools[q.Pool]
		if !ok {
			return nil, &QuotaError{Reason: QuotaPoolNotFound, Pool: q.Pool}
		}

		hasUnits := q.Units != nil
		hasDirect := q.Quota != nil
		switch {
		case hasUnits && hasDirect:
			return nil, &QuotaError{Reason: QuotaModeConflict, Pool: q.Pool}
		case !hasUnits && !hasDirect:
			return nil, &QuotaError{Reason: QuotaModeRequired, Pool: q.Pool}
		case hasDirect:
			if err := validateDirectQuota(q.Pool, q.Quota); err != nil {
				return nil, err
			}
			out = append(out, tenantv1alpha1.QuotaSpec{
				Pool: q.Pool,
				Min:  emptyToNil(copyResourceList(q.Quota.Min)),
				Max:  emptyToNil(copyResourceList(q.Quota.Max)),
			})
			continue
		}

		min := corev1.ResourceList{}
		max := corev1.ResourceList{}
		for _, sel := range q.Units {
			if sel.Quantity < 0 {
				return nil, &QuotaError{Reason: QuotaBadQuantity, Pool: q.Pool, Unit: sel.UnitName}
			}
			unit := findUnit(pool, sel.UnitName)
			if unit == nil {
				return nil, &QuotaError{Reason: QuotaUnitNotFound, Pool: q.Pool, Unit: sel.UnitName}
			}
			addScaled(min, unit.Requests, sel.Quantity)
			addScaled(max, unit.Limits, sel.Quantity)
		}
		out = append(out, tenantv1alpha1.QuotaSpec{
			Pool: q.Pool,
			Min:  emptyToNil(min),
			Max:  emptyToNil(max),
		})
	}
	return out, nil
}

// PoolNames returns the distinct pool names referenced by the selection. Direct
// quota inputs do not need ResourceUnit expansion, but still validate the pool
// exists so Tenant.spec.quotas[].pool remains meaningful.
func PoolNames(selections []Quota) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, q := range selections {
		if _, ok := seen[q.Pool]; ok {
			continue
		}
		seen[q.Pool] = struct{}{}
		out = append(out, q.Pool)
	}
	return out
}

// QuotasToAnnotation JSON-encodes only unit-mode quota selections for the
// round-trip annotation; direct quota inputs are already represented by
// spec.quotas[].min/max. An empty unit selection set clears the annotation.
func QuotasToAnnotation(quotas []Quota) (string, error) {
	unitQuotas := make([]Quota, 0, len(quotas))
	for _, q := range quotas {
		if q.Units == nil {
			continue
		}
		q.Quota = nil
		unitQuotas = append(unitQuotas, q)
	}
	if len(unitQuotas) == 0 {
		return "", nil
	}
	b, err := json.Marshal(unitQuotas)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func quotasFromAnnotation(ann map[string]string) []Quota {
	raw := ann[QuotasAnnotation]
	if raw == "" {
		return []Quota{}
	}
	var out []Quota
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []Quota{}
	}
	return out
}

func quotasToAPI(specs []tenantv1alpha1.QuotaSpec, ann map[string]string) []Quota {
	unitByPool := map[string]Quota{}
	for _, q := range quotasFromAnnotation(ann) {
		if q.Pool == "" || q.Units == nil {
			continue
		}
		q.Quota = nil
		unitByPool[q.Pool] = q
	}

	out := make([]Quota, 0, len(specs))
	for _, spec := range specs {
		if q, ok := unitByPool[spec.Pool]; ok {
			out = append(out, q)
			continue
		}
		// A valid direct quota always carries max (see validateDirectQuota), so a
		// spec entry with no max cannot be direct input — it is a units-mode fold
		// whose round-trip annotation drifted away. Represent it as an empty units
		// selection: it stays re-foldable (an empty set folds cleanly) and never
		// emits an invalid `quota.max: null`.
		if len(spec.Max) == 0 {
			out = append(out, Quota{Pool: spec.Pool, Units: []QuotaUnit{}})
			continue
		}
		out = append(out, Quota{
			Pool: spec.Pool,
			Quota: &QuotaResources{
				Min: copyResourceList(spec.Min),
				Max: copyResourceList(spec.Max),
			},
		})
	}
	return out
}

func tenantStatusToAPI(s tenantv1alpha1.TenantStatus) *TenantStatus {
	if s.Phase == "" && s.ObservedGeneration == 0 && !s.NamespaceReady &&
		s.Message == "" && len(s.Quotas) == 0 {
		return nil
	}
	dto := &TenantStatus{
		ObservedGeneration: s.ObservedGeneration,
		Phase:              string(s.Phase),
		Message:            s.Message,
		NamespaceReady:     s.NamespaceReady,
	}
	for _, q := range s.Quotas {
		dto.Quotas = append(dto.Quotas, QuotaStatus{Pool: q.Pool, Ready: q.Ready, Used: q.Used})
	}
	return dto
}

func tenantAnnotations(user map[string]string, lastModifiedBy, quotaAnno string) map[string]string {
	out := copyMap(user)
	if out == nil {
		out = map[string]string{}
	}
	if lastModifiedBy != "" {
		out[LastModifiedByAnnotation] = lastModifiedBy
	}
	if quotaAnno != "" {
		out[QuotasAnnotation] = quotaAnno
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func findUnit(pool *axismlv1alpha1.ResourcePool, name string) *axismlv1alpha1.ResourceUnit {
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == name {
			return &pool.Spec.Units[i]
		}
	}
	return nil
}

// addScaled adds each entry of src to acc n times. Repeated Add (rather than a
// Value()×n multiply) preserves milli-precision and avoids int64 overflow on
// large memory quantities; n is the unit count, always small.
func addScaled(acc, src corev1.ResourceList, n int) {
	for name, qty := range src {
		cur := acc[name].DeepCopy()
		for i := 0; i < n; i++ {
			cur.Add(qty)
		}
		acc[name] = cur
	}
}

func emptyToNil(rl corev1.ResourceList) corev1.ResourceList {
	if len(rl) == 0 {
		return nil
	}
	return rl
}

func copyResourceList(in corev1.ResourceList) corev1.ResourceList {
	if len(in) == 0 {
		return nil
	}
	out := make(corev1.ResourceList, len(in))
	for name, qty := range in {
		out[name] = qty.DeepCopy()
	}
	return out
}

func validateDirectQuota(pool string, q *QuotaResources) error {
	if q == nil {
		return &QuotaError{Reason: QuotaModeRequired, Pool: pool}
	}
	if len(q.Max) == 0 {
		return &QuotaError{Reason: QuotaMaxRequired, Pool: pool}
	}
	for name, v := range q.Max {
		if v.Sign() < 0 {
			return &QuotaError{Reason: QuotaNegativeResource, Pool: pool, Resource: string(name)}
		}
	}
	for name, v := range q.Min {
		if v.Sign() < 0 {
			return &QuotaError{Reason: QuotaNegativeResource, Pool: pool, Resource: string(name)}
		}
		mv, ok := q.Max[name]
		if !ok {
			return &QuotaError{Reason: QuotaMinWithoutMax, Pool: pool, Resource: string(name)}
		}
		if v.Cmp(mv) > 0 {
			return &QuotaError{Reason: QuotaMinExceedsMax, Pool: pool, Resource: string(name)}
		}
	}
	return nil
}

func isZeroInitResources(ir tenantv1alpha1.InitResources) bool {
	return len(ir.ImagePullSecrets) == 0 && len(ir.Secrets) == 0 &&
		len(ir.ConfigMaps) == 0 && len(ir.ServiceAccounts) == 0 && len(ir.Volumes) == 0
}

// QuotaErrorReason classifies a quota-folding failure for HTTP mapping.
type QuotaErrorReason string

const (
	QuotaPoolNotFound     QuotaErrorReason = "PoolNotFound"
	QuotaUnitNotFound     QuotaErrorReason = "UnitNotFound"
	QuotaBadQuantity      QuotaErrorReason = "BadQuantity"
	QuotaDuplicatePool    QuotaErrorReason = "DuplicatePool"
	QuotaModeConflict     QuotaErrorReason = "ModeConflict"
	QuotaModeRequired     QuotaErrorReason = "ModeRequired"
	QuotaMaxRequired      QuotaErrorReason = "MaxRequired"
	QuotaNegativeResource QuotaErrorReason = "NegativeResource"
	QuotaMinWithoutMax    QuotaErrorReason = "MinWithoutMax"
	QuotaMinExceedsMax    QuotaErrorReason = "MinExceedsMax"
)

// QuotaError is returned by FoldQuotas when the selection cannot be resolved.
type QuotaError struct {
	Reason   QuotaErrorReason
	Pool     string
	Unit     string
	Resource string
}

func (e *QuotaError) Error() string {
	switch e.Reason {
	case QuotaPoolNotFound:
		return "resource pool not found: " + e.Pool
	case QuotaUnitNotFound:
		return "unit not found in pool " + e.Pool + ": " + e.Unit
	case QuotaBadQuantity:
		return "quantity must be >= 0 for unit " + e.Unit + " in pool " + e.Pool
	case QuotaDuplicatePool:
		return "duplicate quota for pool " + e.Pool
	case QuotaModeConflict:
		return "quota for pool " + e.Pool + " must use either units or quota, not both"
	case QuotaModeRequired:
		return "quota for pool " + e.Pool + " must specify either units or quota"
	case QuotaMaxRequired:
		return "quota.max is required for pool " + e.Pool
	case QuotaNegativeResource:
		return "quota resource " + e.Resource + " must be >= 0 in pool " + e.Pool
	case QuotaMinWithoutMax:
		return "quota.min resource " + e.Resource + " must also be present in quota.max for pool " + e.Pool
	case QuotaMinExceedsMax:
		return "quota.min resource " + e.Resource + " exceeds quota.max in pool " + e.Pool
	default:
		return "invalid quota"
	}
}
