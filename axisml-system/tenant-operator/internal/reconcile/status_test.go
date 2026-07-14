package reconcile

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

func TestAggregate_AllReadyTrue(t *testing.T) {
	a := Aggregate{
		Quotas:           []axisml.QuotaStatus{{Pool: "gpu", Ready: true}},
		ImagePullSecrets: []axisml.InitResourceItemStatus{{Ready: true}},
		Secrets:          []axisml.InitResourceItemStatus{{Ready: true}},
		ConfigMaps:       []axisml.InitResourceItemStatus{{Ready: true}},
		ServiceAccounts:  []axisml.InitResourceItemStatus{{Ready: true}},
	}
	if !a.AllInitResourcesReady() {
		t.Error("AllInitResourcesReady returned false on fully-ready aggregate")
	}
	if !a.AllQuotasReady() {
		t.Error("AllQuotasReady returned false on fully-ready aggregate")
	}
}

func TestAggregate_NotReady(t *testing.T) {
	cases := []struct {
		name string
		a    Aggregate
	}{
		{"image pull secret not ready", Aggregate{ImagePullSecrets: []axisml.InitResourceItemStatus{{Ready: false}}}},
		{"secret not ready", Aggregate{Secrets: []axisml.InitResourceItemStatus{{Ready: false}}}},
		{"configmap not ready", Aggregate{ConfigMaps: []axisml.InitResourceItemStatus{{Ready: false}}}},
		{"sa not ready", Aggregate{ServiceAccounts: []axisml.InitResourceItemStatus{{Ready: false}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.a.AllInitResourcesReady() {
				t.Error("expected AllInitResourcesReady=false")
			}
		})
	}

	a := Aggregate{Quotas: []axisml.QuotaStatus{{Ready: false}}}
	if a.AllQuotasReady() {
		t.Error("expected AllQuotasReady=false")
	}
}

func TestDerivePhase_CriticalFailure(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{CriticalFailure: true, FailureMessage: "boom"}
	phase, msg := DerivePhase(tnt, a, axisml.TenantPhaseActive)
	if phase != axisml.TenantPhaseFailed {
		t.Errorf("phase = %q; want Failed", phase)
	}
	if msg != "boom" {
		t.Errorf("msg = %q; want boom", msg)
	}
}

func TestDerivePhase_AllReady_Active(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{
		NamespaceReady: true,
		Quotas:         []axisml.QuotaStatus{{Ready: true}},
		Secrets:        []axisml.InitResourceItemStatus{{Ready: true}},
	}
	phase, msg := DerivePhase(tnt, a, "")
	if phase != axisml.TenantPhaseActive {
		t.Errorf("phase = %q; want Active", phase)
	}
	if msg != "" {
		t.Errorf("msg = %q; want empty", msg)
	}
}

func TestDerivePhase_TransientNamespaceNotReady(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{NamespaceReady: false, NamespaceMsg: "phase=Terminating"}
	phase, msg := DerivePhase(tnt, a, axisml.TenantPhaseActive)
	if phase != axisml.TenantPhaseActive {
		t.Errorf("transient: phase should be preserved as Active; got %q", phase)
	}
	if msg == "" || msg == "phase=Terminating" {
		t.Errorf("expected wrapped msg containing namespace prefix; got %q", msg)
	}
}

func TestDerivePhase_TransientQuotasNotReady(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{NamespaceReady: true, Quotas: []axisml.QuotaStatus{{Ready: false}}}
	phase, msg := DerivePhase(tnt, a, axisml.TenantPhaseActive)
	if phase != axisml.TenantPhaseActive {
		t.Errorf("phase should remain Active; got %q", phase)
	}
	if msg != "quotas not ready" {
		t.Errorf("msg = %q", msg)
	}
}

func TestDerivePhase_TransientInitResourcesNotReady(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{
		NamespaceReady: true,
		Secrets:        []axisml.InitResourceItemStatus{{Ready: false}},
	}
	phase, msg := DerivePhase(tnt, a, axisml.TenantPhaseActive)
	if phase != axisml.TenantPhaseActive {
		t.Errorf("phase should remain Active; got %q", phase)
	}
	if msg != "init resources not ready" {
		t.Errorf("msg = %q", msg)
	}
}

func TestDerivePhase_NoPreviousPhaseStaysEmpty(t *testing.T) {
	tnt := &axisml.Tenant{}
	a := Aggregate{NamespaceReady: false, NamespaceMsg: "creating"}
	phase, _ := DerivePhase(tnt, a, "")
	if phase != "" {
		t.Errorf("brand-new tenant: phase should remain empty; got %q", phase)
	}
}

func TestBuildStatus_Conditions(t *testing.T) {
	tnt := &axisml.Tenant{ObjectMeta: metav1.ObjectMeta{Generation: 7}}
	a := Aggregate{
		NamespaceReady: true,
		Quotas:         []axisml.QuotaStatus{{Pool: "gpu", Ready: true}},
		Secrets:        []axisml.InitResourceItemStatus{{Name: "x", Ready: true}},
	}
	st := BuildStatus(tnt, a, axisml.TenantPhaseActive, "")
	if st.ObservedGeneration != 7 {
		t.Errorf("ObservedGeneration = %d; want 7", st.ObservedGeneration)
	}
	if !st.NamespaceReady {
		t.Error("NamespaceReady should be true")
	}
	if len(st.Conditions) != 3 {
		t.Errorf("expected 3 base conditions; got %d", len(st.Conditions))
	}
	conds := condsByType(st.Conditions)
	if conds[axisml.ConditionNamespaceReady].Status != metav1.ConditionTrue {
		t.Error("NamespaceReady condition should be True")
	}
	if conds[axisml.ConditionQuotasReady].Status != metav1.ConditionTrue {
		t.Error("QuotasReady condition should be True")
	}
	if conds[axisml.ConditionInitResourcesReady].Status != metav1.ConditionTrue {
		t.Error("InitResourcesReady condition should be True")
	}
}

func TestBuildStatus_FailedPhaseAddsFailedCondition(t *testing.T) {
	tnt := &axisml.Tenant{ObjectMeta: metav1.ObjectMeta{Generation: 1}}
	a := Aggregate{CriticalFailure: true, FailureMessage: "namespace gone"}
	st := BuildStatus(tnt, a, axisml.TenantPhaseFailed, "namespace gone")
	conds := condsByType(st.Conditions)
	c, ok := conds[axisml.ConditionFailed]
	if !ok {
		t.Fatal("expected Failed condition")
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("Failed condition status = %v", c.Status)
	}
	if c.Message != "namespace gone" {
		t.Errorf("Failed message = %q", c.Message)
	}
}

func TestBuildStatus_PreservesLastTransitionTime(t *testing.T) {
	old := metav1.Time{Time: time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)}
	tnt := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{Generation: 1},
		Status: axisml.TenantStatus{
			Conditions: []metav1.Condition{{
				Type:               axisml.ConditionNamespaceReady,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: old,
			}},
		},
	}
	a := Aggregate{NamespaceReady: true}
	st := BuildStatus(tnt, a, axisml.TenantPhaseActive, "")
	c := condsByType(st.Conditions)[axisml.ConditionNamespaceReady]
	if !c.LastTransitionTime.Equal(&old) {
		t.Errorf("LastTransitionTime advanced when status unchanged: %v vs %v", c.LastTransitionTime, old)
	}
}

func TestBuildStatus_AdvancesLastTransitionOnFlip(t *testing.T) {
	old := metav1.Time{Time: time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)}
	tnt := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{Generation: 2},
		Status: axisml.TenantStatus{
			Conditions: []metav1.Condition{{
				Type:               axisml.ConditionNamespaceReady,
				Status:             metav1.ConditionFalse,
				LastTransitionTime: old,
			}},
		},
	}
	a := Aggregate{NamespaceReady: true}
	st := BuildStatus(tnt, a, axisml.TenantPhaseActive, "")
	c := condsByType(st.Conditions)[axisml.ConditionNamespaceReady]
	if !c.LastTransitionTime.After(old.Time) {
		t.Errorf("LastTransitionTime did not advance on flip: got %v", c.LastTransitionTime)
	}
}

func condsByType(in []metav1.Condition) map[string]metav1.Condition {
	out := make(map[string]metav1.Condition, len(in))
	for _, c := range in {
		out[c.Type] = c
	}
	return out
}
