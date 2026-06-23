package server

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// TenantToAPI renders a Tenant CR into its REST representation. Quotas come from the
// round-trip annotation (business form); status is read live from the CR.
func TenantToAPI(t *tenantv1alpha1.Tenant) Tenant {
	dto := Tenant{
		Name:            t.Name,
		Namespace:       t.Spec.Namespace,
		Quotas:          quotasFromAnnotation(t.Annotations),
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

// FoldQuotas converts the business-form selection (`unit × quantity` per pool)
// into ElasticQuota min/max by summing each unit's requests/limits scaled by
// its quantity, resolving units against the supplied ResourcePools. The result
// is written 1:1 to Tenant.spec.quotas[] for tenant-operator to render.
func FoldQuotas(selections []Quota, pools map[string]*axismlv1alpha1.ResourcePool) ([]tenantv1alpha1.QuotaSpec, error) {
	out := make([]tenantv1alpha1.QuotaSpec, 0, len(selections))
	for _, q := range selections {
		pool, ok := pools[q.Pool]
		if !ok {
			return nil, &QuotaError{Reason: QuotaPoolNotFound, Pool: q.Pool}
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
			Name: q.Pool,
			Min:  emptyToNil(min),
			Max:  emptyToNil(max),
		})
	}
	return out, nil
}

// PoolNames returns the distinct pool names referenced by the selection, so
// the handler can fetch exactly the ResourcePools it needs for folding.
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

// QuotasToAnnotation JSON-encodes the business-form selection for the
// round-trip annotation; an empty selection clears it.
func QuotasToAnnotation(quotas []Quota) (string, error) {
	if len(quotas) == 0 {
		return "", nil
	}
	b, err := json.Marshal(quotas)
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

func isZeroInitResources(ir tenantv1alpha1.InitResources) bool {
	return len(ir.ImagePullSecrets) == 0 && len(ir.Secrets) == 0 &&
		len(ir.ConfigMaps) == 0 && len(ir.ServiceAccounts) == 0
}

// QuotaErrorReason classifies a quota-folding failure for HTTP mapping.
type QuotaErrorReason string

const (
	QuotaPoolNotFound QuotaErrorReason = "PoolNotFound"
	QuotaUnitNotFound QuotaErrorReason = "UnitNotFound"
	QuotaBadQuantity  QuotaErrorReason = "BadQuantity"
)

// QuotaError is returned by FoldQuotas when the selection cannot be resolved.
type QuotaError struct {
	Reason QuotaErrorReason
	Pool   string
	Unit   string
}

func (e *QuotaError) Error() string {
	switch e.Reason {
	case QuotaPoolNotFound:
		return "resource pool not found: " + e.Pool
	case QuotaUnitNotFound:
		return "unit not found in pool " + e.Pool + ": " + e.Unit
	case QuotaBadQuantity:
		return "quantity must be >= 0 for unit " + e.Unit + " in pool " + e.Pool
	default:
		return "invalid quota"
	}
}
