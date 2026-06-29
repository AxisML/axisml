package reconcile

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

func TestPerTenantResourceName(t *testing.T) {
	cases := []struct {
		tenant string
		sub    string
		want   string
	}{
		{"team-a", "registry", "axisml-tenant-team-a-registry"},
		{"team-a", "", "axisml-tenant-team-a-"},
		{"alpha", "creds", "axisml-tenant-alpha-creds"},
	}
	for _, tc := range cases {
		got := PerTenantResourceName(tc.tenant, tc.sub)
		if got != tc.want {
			t.Errorf("PerTenantResourceName(%q, %q) = %q; want %q", tc.tenant, tc.sub, got, tc.want)
		}
	}
}

func TestElasticQuotaName(t *testing.T) {
	got := ElasticQuotaName("team-a", "gpu", "default")
	want := "axisml-team-a-gpu-default"
	if got != want {
		t.Errorf("ElasticQuotaName = %q; want %q", got, want)
	}
}

func TestTenantLabels(t *testing.T) {
	tnt := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-a",
			Labels: map[string]string{axisml.LabelTenantID: "uuid-1"},
		},
	}
	got := TenantLabels(tnt)
	if got[axisml.LabelTenantID] != "uuid-1" {
		t.Errorf("TenantID label = %q; want uuid-1", got[axisml.LabelTenantID])
	}
	if got[axisml.LabelManagedBy] != axisml.ManagedByValue {
		t.Errorf("ManagedBy label = %q; want %q", got[axisml.LabelManagedBy], axisml.ManagedByValue)
	}
}

func TestApplyTenantLabels_NilMap(t *testing.T) {
	tnt := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-a",
			Labels: map[string]string{axisml.LabelTenantID: "uuid-1"},
		},
	}
	got := ApplyTenantLabels(tnt, nil)
	if got == nil {
		t.Fatal("ApplyTenantLabels(nil) returned nil")
	}
	if got[axisml.LabelTenantID] != "uuid-1" {
		t.Errorf("missing tenant id; got %v", got)
	}
}

func TestApplyTenantLabels_PreservesUserKeys(t *testing.T) {
	tnt := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "team-a",
			Labels: map[string]string{axisml.LabelTenantID: "uuid-1"},
		},
	}
	in := map[string]string{"axisml.io/secret-role": "imagepull"}
	got := ApplyTenantLabels(tnt, in)
	if got["axisml.io/secret-role"] != "imagepull" {
		t.Errorf("user key dropped: %v", got)
	}
	if got[axisml.LabelTenantID] != "uuid-1" {
		t.Errorf("tenant id missing: %v", got)
	}
}
