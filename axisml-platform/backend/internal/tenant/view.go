package tenant

import (
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
)

// QuotaUnitSpec is one resource-unit count within a pool quota.
type QuotaUnitSpec struct {
	UnitName string
	Quantity int
}

// QuotaSpec is a tenant's quota in one pool.
type QuotaSpec struct {
	Pool  string
	Units []QuotaUnitSpec
}

// buildView merges the durable record with the live Tenant CR (cr may be nil
// when cluster-manager is unreachable) into the contract Tenant view.
func buildView(row *store.Tenant, cr *clustermanager.Tenant) *server.Tenant {
	t := &server.Tenant{
		Identifier:          row.Identifier,
		KubernetesNamespace: row.KubernetesNamespace,
		DisplayName:         row.DisplayName,
		Description:         row.Description,
		Owner:               row.Owner,
		Labels:              server.StringMap(row.Labels),
		Annotations:         server.StringMap(row.Annotations),
		Suspended:           row.SuspendedAt != nil,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	phase := server.TenantPhase("Creating")
	if cr != nil {
		t.Quotas = mapQuotas(cr.Quotas)
		if cr.Status != nil {
			st := server.TenantStatus{}
			if cr.Status.Message != nil {
				st.Message = *cr.Status.Message
			}
			st.Quotas = mapQuotaStatuses(cr.Status.Quotas)
			t.Status = st
			if cr.Status.Phase != nil && *cr.Status.Phase != "" {
				phase = server.TenantPhase(*cr.Status.Phase)
			} else {
				phase = "Active"
			}
		} else if cr.Phase != nil && *cr.Phase != "" {
			phase = server.TenantPhase(*cr.Phase)
		} else {
			phase = "Active"
		}
	}
	if row.SuspendedAt != nil {
		phase = "Suspended"
	}
	t.Phase = phase
	return t
}

func mapQuotas(qs []clustermanager.Quota) []server.Quota {
	if len(qs) == 0 {
		return nil
	}
	out := make([]server.Quota, 0, len(qs))
	for _, q := range qs {
		units := make([]server.QuotaUnit, 0, len(q.Units))
		for _, u := range q.Units {
			units = append(units, server.QuotaUnit{UnitName: u.UnitName, Quantity: u.Quantity})
		}
		out = append(out, server.Quota{Pool: q.Pool, Units: units})
	}
	return out
}

func mapQuotaStatuses(qs *[]clustermanager.QuotaStatus) []server.QuotaStatus {
	if qs == nil || len(*qs) == 0 {
		return nil
	}
	out := make([]server.QuotaStatus, 0, len(*qs))
	for _, q := range *qs {
		// cluster-manager reports per-pool usage as Kubernetes resource quantities
		// (Used), not unit counts; surface the pool readiness here and leave the
		// per-unit usage projection for a later round.
		out = append(out, server.QuotaStatus{Pool: q.Pool})
	}
	return out
}
