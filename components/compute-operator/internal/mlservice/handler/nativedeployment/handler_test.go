package nativedeployment

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	"github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
)

// minimalSpec returns a spec that the (native, deployment) handler accepts.
// Tests mutate copies of this to exercise individual validation paths.
func minimalSpec() *axisml.MLServiceSpec {
	return &axisml.MLServiceSpec{
		Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
		Scheduling: axisml.Scheduling{Quota: "axisml-demo-default-training"},
		Roles: []axisml.RoleSpec{{
			Name:     axisml.DefaultRoleName,
			Replicas: 1,
			Template: axisml.PodTemplate{
				Image: "nginx:1.27",
				Ports: []axisml.PodPort{{Name: "http", ContainerPort: 8080}},
			},
		}},
	}
}

func TestValidate_AcceptsMinimalSpec(t *testing.T) {
	h := &Handler{}
	if v := h.Validate(minimalSpec()); !v.OK() {
		t.Fatalf("expected validation to pass; got errors: %v", v.Errors)
	}
}

func TestValidate_RejectsMultipleRoles(t *testing.T) {
	spec := minimalSpec()
	spec.Roles = append(spec.Roles, spec.Roles[0])
	h := &Handler{}
	v := h.Validate(spec)
	if v.OK() {
		t.Fatal("expected validation to fail with two roles")
	}
	if !containsSubstring(v.Errors, "exactly one role") {
		t.Errorf("expected 'exactly one role' in errors; got %v", v.Errors)
	}
}

func TestValidate_RejectsWrongRoleName(t *testing.T) {
	spec := minimalSpec()
	spec.Roles[0].Name = "transformer"
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail when role name is not predictor")
	}
}

func TestValidate_RejectsRouteWithUnknownPortName(t *testing.T) {
	spec := minimalSpec()
	spec.Route = &axisml.Route{Enabled: true, PortName: "grpc"}
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail when route.portName is unknown")
	}
}

func TestValidate_RejectsDeferredAuthAndWarnsOnTrafficPolicy(t *testing.T) {
	spec := minimalSpec()
	spec.Route = &axisml.Route{
		Enabled:   true,
		Auth:      &axisml.RouteAuth{Type: axisml.RouteAuthJWT},
		RateLimit: &axisml.RouteRateLimit{RequestsPerSecond: 10},
	}
	h := &Handler{}
	v := h.Validate(spec)
	if v.OK() {
		t.Fatal("expected validation to reject auth until SecurityPolicy is implemented")
	}
	if len(v.Warnings) < 1 {
		t.Errorf("expected a rateLimit warning; got %v", v.Warnings)
	}
}

func TestBuildDeployment_InjectsRequiredLabels(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "smoke",
			Namespace: "tenant-demo",
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-1",
				axisml.LabelTenant:    "demo",
				axisml.LabelQuota:     "training",
			},
		},
		Spec: *minimalSpec(),
	}
	dep := buildDeployment(mls)

	if got := dep.Spec.Template.Spec.SchedulerName; got != axisml.SchedulerName {
		t.Errorf("schedulerName = %q; want %q", got, axisml.SchedulerName)
	}

	wantLabels := map[string]string{
		axisml.LabelServiceID:      "uuid-1",
		axisml.LabelRole:           "predictor",
		axisml.LabelKoordQuotaName: "axisml-demo-default-training",
		axisml.LabelQuota:          "training",
		axisml.LabelTenant:         "demo",
	}
	for k, want := range wantLabels {
		if got := dep.Spec.Template.Labels[k]; got != want {
			t.Errorf("pod label %q = %q; want %q", k, got, want)
		}
	}

	// Selector must NOT carry the quota / tenant labels — they evolve and
	// would break Service routing if pinned into the selector.
	if _, ok := dep.Spec.Selector.MatchLabels[axisml.LabelKoordQuotaName]; ok {
		t.Error("selector unexpectedly contains koord quota label")
	}
}

func TestBuildService_TargetPortMatchesContainerPort(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	svc := buildService(mls)
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 service port; got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].TargetPort.IntValue() != 8080 {
		t.Errorf("targetPort = %v; want 8080", svc.Spec.Ports[0].TargetPort)
	}
}

func TestMapStatus_PhasePending_WhenNoDeployment(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	upd := mapStatus(handler.Snapshot{Service: mls})
	if upd.Phase != axisml.PhasePending {
		t.Errorf("phase = %s; want Pending", upd.Phase)
	}
}

func TestMapStatus_PhaseReady_WhenDeploymentMatches(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{dep, svc},
	})
	if upd.Phase != axisml.PhaseReady {
		t.Errorf("phase = %s; want Ready", upd.Phase)
	}
	if upd.Endpoint != "smoke.tenant-demo.svc.cluster.local:8080" {
		t.Errorf("endpoint = %q; want internal DNS form", upd.Endpoint)
	}
	if upd.ReadyReplicas != 1 {
		t.Errorf("readyReplicas = %d; want 1", upd.ReadyReplicas)
	}
}

func TestMapStatus_PhaseDegraded_WhenPartialReady(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	mls.Spec.Roles[0].Replicas = 3
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{dep},
	})
	if upd.Phase != axisml.PhaseDegraded {
		t.Errorf("phase = %s; want Degraded", upd.Phase)
	}
}

func TestMapStatus_PhaseFailed_OnProgressDeadlineExceeded(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{{
				Type:   appsv1.DeploymentProgressing,
				Status: corev1.ConditionFalse,
				Reason: "ProgressDeadlineExceeded",
			}},
		},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{dep},
	})
	if upd.Phase != axisml.PhaseFailed {
		t.Errorf("phase = %s; want Failed", upd.Phase)
	}
}

func TestMapStatus_RouteEnabled_DegradedWhenNotAccepted(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	mls.Spec.Route = &axisml.Route{Enabled: true, Hostname: "demo.example.com"}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{dep, svc, route},
	})
	if upd.Phase != axisml.PhaseDegraded {
		t.Errorf("phase = %s; want Degraded (Deployment ready but HTTPRoute not Accepted)", upd.Phase)
	}
	if !strings.Contains(upd.Endpoint, "svc.cluster.local") {
		t.Errorf("endpoint = %q; want fallback to internal DNS", upd.Endpoint)
	}
}

func TestMapStatus_RouteEnabled_ReadyWithExternalEndpoint(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	mls.Spec.Route = &axisml.Route{Enabled: true, Hostname: "demo.example.com", Path: "/predict"}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status: gwapiv1.HTTPRouteStatus{
			RouteStatus: gwapiv1.RouteStatus{
				Parents: []gwapiv1.RouteParentStatus{{
					Conditions: []metav1.Condition{{
						Type:   string(gwapiv1.RouteConditionAccepted),
						Status: metav1.ConditionTrue,
					}},
				}},
			},
		},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{dep, svc, route},
	})
	if upd.Phase != axisml.PhaseReady {
		t.Errorf("phase = %s; want Ready", upd.Phase)
	}
	if upd.Endpoint != "https://demo.example.com/predict" {
		t.Errorf("endpoint = %q; want external URL", upd.Endpoint)
	}
}

func containsSubstring(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
