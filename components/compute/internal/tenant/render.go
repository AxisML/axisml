package tenant

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	tenantv1alpha1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"

	"github.com/axisml/axisml/components/compute/internal/quota"
	"github.com/axisml/axisml/components/compute/internal/resourcepool"
	"github.com/axisml/axisml/components/compute/internal/spechash"
)

// SpecSnapshot is the JSON shape persisted in tenants.spec. It mirrors the
// fields of tenantv1alpha1.TenantSpec but stores quotas[] as a snapshot;
// the live quotas[] used at patch time is rendered fresh from PG.
type SpecSnapshot struct {
	DisplayName   string                       `json:"displayName,omitempty"`
	Annotations   map[string]string            `json:"annotations,omitempty"`
	Namespace     tenantv1alpha1.NamespaceSpec `json:"namespace"`
	Quotas        []tenantv1alpha1.QuotaSpec   `json:"quotas,omitempty"`
	InitResources tenantv1alpha1.InitResources `json:"initResources,omitempty"`
	Suspended     bool                         `json:"suspended,omitempty"`
}

// EncodeSpec serializes a SpecSnapshot for PG storage.
func EncodeSpec(s SpecSnapshot) ([]byte, error) {
	return json.Marshal(s)
}

// DecodeSpec deserializes a stored snapshot.
func DecodeSpec(b []byte) (SpecSnapshot, error) {
	var s SpecSnapshot
	if len(b) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

// RenderQuotas materialises Tenant.spec.quotas[] from the live quotas table.
// Pass tx=nil to use the service's default DB.
func RenderQuotas(ctx context.Context, tx *gorm.DB, quotas *quota.Service, pools *resourcepool.Service, tenantID uuid.UUID) ([]tenantv1alpha1.QuotaSpec, error) {
	rows, err := quotas.ListByTenantTx(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]tenantv1alpha1.QuotaSpec, 0, len(rows))
	for i := range rows {
		row := rows[i]
		var qs quota.Spec
		if len(row.Spec) > 0 {
			if err := json.Unmarshal(row.Spec, &qs); err != nil {
				return nil, err
			}
		}
		pool, err := pools.GetByID(ctx, row.PoolID)
		if err != nil {
			return nil, err
		}
		out = append(out, tenantv1alpha1.QuotaSpec{
			Pool: pool.Name,
			Name: row.Name,
			Min:  qs.Min,
			Max:  qs.Max,
		})
	}
	return out, nil
}

// ComputeDesiredHash renders the live spec snapshot (with quotas) and returns
// its canonical SHA-256 hex digest. Used by the tenant reconciler before patch.
func ComputeDesiredHash(snap SpecSnapshot) (string, error) {
	return spechash.Compute(snap)
}
