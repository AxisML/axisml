package server

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	axismlv1alpha1 "github.com/axisml/axisml/axisml-system/cluster-manager/api/v1alpha1"
)

func testPool() *axismlv1alpha1.ResourcePool {
	p := &axismlv1alpha1.ResourcePool{}
	p.Name = "p1"
	p.Spec.Units = []axismlv1alpha1.ResourceUnit{{
		Name: "cpu-small",
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1"),
			"memory": resource.MustParse("2Gi"),
		},
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("2"),
			"memory": resource.MustParse("4Gi"),
		},
	}}
	return p
}

func TestFoldQuotas_SumsUnitsByQuantity(t *testing.T) {
	pools := map[string]*axismlv1alpha1.ResourcePool{"p1": testPool()}
	sel := []Quota{{Pool: "p1", Units: []QuotaUnit{{UnitName: "cpu-small", Quantity: 3}}}}

	folded, err := FoldQuotas(sel, pools)
	if err != nil {
		t.Fatalf("FoldQuotas: %v", err)
	}
	if len(folded) != 1 {
		t.Fatalf("want 1 quota, got %d", len(folded))
	}
	q := folded[0]
	if got := q.Min["cpu"]; got.String() != "3" {
		t.Errorf("min cpu = %s, want 3", got.String())
	}
	if got := q.Max["cpu"]; got.String() != "6" {
		t.Errorf("max cpu = %s, want 6", got.String())
	}
	if got := q.Min["memory"]; got.String() != "6Gi" {
		t.Errorf("min memory = %s, want 6Gi", got.String())
	}
	if got := q.Max["memory"]; got.String() != "12Gi" {
		t.Errorf("max memory = %s, want 12Gi", got.String())
	}
	if q.Pool != "p1" || q.Name != "p1" {
		t.Errorf("pool/name = %s/%s, want p1/p1", q.Pool, q.Name)
	}
}

func TestFoldQuotas_Errors(t *testing.T) {
	pools := map[string]*axismlv1alpha1.ResourcePool{"p1": testPool()}

	cases := []struct {
		name   string
		sel    []Quota
		reason QuotaErrorReason
	}{
		{"unknown pool", []Quota{{Pool: "ghost", Units: []QuotaUnit{{UnitName: "cpu-small", Quantity: 1}}}}, QuotaPoolNotFound},
		{"unknown unit", []Quota{{Pool: "p1", Units: []QuotaUnit{{UnitName: "ghost", Quantity: 1}}}}, QuotaUnitNotFound},
		{"bad quantity", []Quota{{Pool: "p1", Units: []QuotaUnit{{UnitName: "cpu-small", Quantity: -1}}}}, QuotaBadQuantity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := FoldQuotas(tc.sel, pools)
			qe, ok := err.(*QuotaError)
			if !ok {
				t.Fatalf("want *QuotaError, got %T (%v)", err, err)
			}
			if qe.Reason != tc.reason {
				t.Errorf("reason = %s, want %s", qe.Reason, tc.reason)
			}
		})
	}
}

func TestQuotaAnnotationRoundTrip(t *testing.T) {
	sel := []Quota{{Pool: "p1", Units: []QuotaUnit{{UnitName: "cpu-small", Quantity: 3}}}}
	anno, err := QuotasToAnnotation(sel)
	if err != nil {
		t.Fatalf("QuotasToAnnotation: %v", err)
	}
	got := quotasFromAnnotation(map[string]string{QuotasAnnotation: anno})
	if len(got) != 1 || got[0].Pool != "p1" || got[0].Units[0].Quantity != 3 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	// Empty selection clears the annotation.
	if anno, _ := QuotasToAnnotation(nil); anno != "" {
		t.Errorf("empty selection should yield empty annotation, got %q", anno)
	}
}
