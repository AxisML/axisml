package reconcile

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
)

// Aggregate carries per-subreconciler outcomes for one Reconcile pass.
type Aggregate struct {
	NamespaceReady bool
	NamespaceMsg   string

	Quotas []axisml.QuotaStatus

	ImagePullSecrets []axisml.InitResourceItemStatus
	Secrets          []axisml.InitResourceItemStatus
	ConfigMaps       []axisml.InitResourceItemStatus
	ServiceAccounts  []axisml.InitResourceItemStatus

	// CriticalFailure indicates a non-transient failure on Namespace or
	// ElasticQuota that should drive phase=Failed (design §4 row 3).
	CriticalFailure bool
	FailureMessage  string
}

// AllInitResourcesReady returns true when every per-tenant init resource
// reconciled OK on this pass.
func (a Aggregate) AllInitResourcesReady() bool {
	for _, s := range a.ImagePullSecrets {
		if !s.Ready {
			return false
		}
	}
	for _, s := range a.Secrets {
		if !s.Ready {
			return false
		}
	}
	for _, s := range a.ConfigMaps {
		if !s.Ready {
			return false
		}
	}
	for _, s := range a.ServiceAccounts {
		if !s.Ready {
			return false
		}
	}
	return true
}

// AllQuotasReady is true when every quota's CR was reconciled OK on this pass.
func (a Aggregate) AllQuotasReady() bool {
	for _, q := range a.Quotas {
		if !q.Ready {
			return false
		}
	}
	return true
}

// DerivePhase implements design §4's phase derivation table. The previous
// phase is supplied so transient-progress paths can preserve state. Callers
// must short-circuit Suspended themselves before invoking this — suspension
// is a phase-only signal and shouldn't go through the readiness derivation.
func DerivePhase(t *axisml.Tenant, a Aggregate, previous axisml.TenantPhase) (axisml.TenantPhase, string) {
	if a.CriticalFailure {
		return axisml.TenantPhaseFailed, a.FailureMessage
	}
	if a.NamespaceReady && a.AllQuotasReady() && a.AllInitResourcesReady() {
		return axisml.TenantPhaseActive, ""
	}
	// Transient progress: keep the previous phase and surface the most
	// informative message. For a brand-new tenant (no previous phase) that
	// hasn't reached readiness yet, leave the phase empty rather than
	// defaulting to Active — Compute treats Active as usable and would admit
	// workloads before required tenant resources exist.
	msg := a.FailureMessage
	if msg == "" {
		switch {
		case !a.NamespaceReady:
			msg = "namespace not ready: " + a.NamespaceMsg
		case !a.AllQuotasReady():
			msg = "quotas not ready"
		case !a.AllInitResourcesReady():
			msg = "init resources not ready"
		}
	}
	return previous, msg
}

// BuildStatus assembles a TenantStatus from the aggregate.
func BuildStatus(t *axisml.Tenant, a Aggregate, phase axisml.TenantPhase, message string) axisml.TenantStatus {
	st := axisml.TenantStatus{
		ObservedGeneration: t.Generation,
		Phase:              phase,
		Message:            message,
		NamespaceReady:     a.NamespaceReady,
		Quotas:             a.Quotas,
		InitResources: axisml.InitResourcesStatus{
			ImagePullSecrets: a.ImagePullSecrets,
			Secrets:          a.Secrets,
			ConfigMaps:       a.ConfigMaps,
			ServiceAccounts:  a.ServiceAccounts,
		},
	}

	prev := t.Status.Conditions
	st.Conditions = []metav1.Condition{
		condition(prev, axisml.ConditionNamespaceReady, a.NamespaceReady, "NamespaceReconciled", a.NamespaceMsg, t.Generation),
		condition(prev, axisml.ConditionQuotasReady, a.AllQuotasReady(), "QuotasReconciled", "", t.Generation),
		condition(prev, axisml.ConditionInitResourcesReady, a.AllInitResourcesReady(), "InitResourcesReconciled", "", t.Generation),
	}
	if t.Spec.Suspended {
		st.Conditions = append(st.Conditions, condition(prev, axisml.ConditionSuspended, true, "TenantSuspended", "spec.suspended=true", t.Generation))
	}
	if phase == axisml.TenantPhaseFailed {
		st.Conditions = append(st.Conditions, condition(prev, axisml.ConditionFailed, true, "ReconcileFailed", message, t.Generation))
	}
	return st
}

// condition builds a metav1.Condition while preserving LastTransitionTime
// from prev when the status hasn't actually flipped. Without this, every
// reconcile would rewrite the timestamp and trigger another status write.
func condition(prev []metav1.Condition, t string, ok bool, reason, message string, gen int64) metav1.Condition {
	status := metav1.ConditionTrue
	if !ok {
		status = metav1.ConditionFalse
	}
	transition := metav1.Now()
	for i := range prev {
		if prev[i].Type == t {
			if prev[i].Status == status && !prev[i].LastTransitionTime.IsZero() {
				transition = prev[i].LastTransitionTime
			}
			break
		}
	}
	return metav1.Condition{
		Type:               t,
		Status:             status,
		ObservedGeneration: gen,
		LastTransitionTime: transition,
		Reason:             reason,
		Message:            message,
	}
}
