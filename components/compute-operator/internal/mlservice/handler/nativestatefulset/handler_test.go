package nativestatefulset

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	"github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
)

// minimalSpec returns a spec the (native, statefulset) handler accepts.
func minimalSpec() *axisml.MLServiceSpec {
	return &axisml.MLServiceSpec{
		Backend:    axisml.Backend{Name: "native", Engine: "statefulset"},
		Scheduling: axisml.Scheduling{Quota: "axisml-demo-default-training"},
		ModelRef:   axisml.ModelRef{Name: "dummy", Version: "v1"},
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

func setConfig(spec *axisml.MLServiceSpec, raw string) {
	spec.Backend.Config = &runtime.RawExtension{Raw: []byte(raw)}
}

func TestValidate_AcceptsMinimalSpec(t *testing.T) {
	h := &Handler{}
	if v := h.Validate(minimalSpec()); !v.OK() {
		t.Fatalf("expected validation to pass; got errors: %v", v.Errors)
	}
}

func TestValidate_AcceptsValidConfig(t *testing.T) {
	spec := minimalSpec()
	setConfig(spec, `{"podManagementPolicy":"Parallel","serviceName":"foo"}`)
	h := &Handler{}
	if v := h.Validate(spec); !v.OK() {
		t.Fatalf("expected validation to pass; got errors: %v", v.Errors)
	}
}

func TestValidate_RejectsUnknownConfigKey(t *testing.T) {
	spec := minimalSpec()
	setConfig(spec, `{"volumeClaimTemplates":[]}`)
	h := &Handler{}
	v := h.Validate(spec)
	if v.OK() {
		t.Fatal("expected validation to fail when unknown backend.config key is set")
	}
	if !containsSubstring(v.Errors, "backend.config invalid") {
		t.Errorf("expected 'backend.config invalid' in errors; got %v", v.Errors)
	}
}

func TestValidate_RejectsBadPodManagementPolicy(t *testing.T) {
	spec := minimalSpec()
	setConfig(spec, `{"podManagementPolicy":"Bogus"}`)
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail for invalid podManagementPolicy")
	}
}

func TestValidate_RejectsBadServiceName(t *testing.T) {
	spec := minimalSpec()
	setConfig(spec, `{"serviceName":"NOT_DNS"}`)
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail for non-DNS-1123 serviceName")
	}
}

func TestValidate_RejectsMultipleRoles(t *testing.T) {
	spec := minimalSpec()
	spec.Roles = append(spec.Roles, spec.Roles[0])
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail with two roles")
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

func TestValidate_RejectsMissingModelRef(t *testing.T) {
	spec := minimalSpec()
	spec.ModelRef = axisml.ModelRef{}
	h := &Handler{}
	if v := h.Validate(spec); v.OK() {
		t.Fatal("expected validation to fail with empty modelRef")
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

func TestValidate_WarnsOnDeferredPolicies(t *testing.T) {
	spec := minimalSpec()
	spec.Route = &axisml.Route{
		Enabled:   true,
		Auth:      &axisml.RouteAuth{Type: axisml.RouteAuthJWT},
		RateLimit: &axisml.RouteRateLimit{RequestsPerSecond: 10},
	}
	h := &Handler{}
	v := h.Validate(spec)
	if !v.OK() {
		t.Fatalf("expected validation to pass with warnings; got errors %v", v.Errors)
	}
	if len(v.Warnings) < 2 {
		t.Errorf("expected ≥2 warnings; got %v", v.Warnings)
	}
}

func TestValidate_WarnsOnProgressDeadlineSeconds(t *testing.T) {
	spec := minimalSpec()
	pds := int32(60)
	spec.RunPolicy.ProgressDeadlineSeconds = &pds
	h := &Handler{}
	v := h.Validate(spec)
	if !v.OK() {
		t.Fatalf("expected validation to pass with warning; got errors %v", v.Errors)
	}
	if !containsSubstring(v.Warnings, "progressDeadlineSeconds") {
		t.Errorf("expected progressDeadlineSeconds warning; got %v", v.Warnings)
	}
}

func TestBuildStatefulSet_InjectsRequiredLabels(t *testing.T) {
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
	sts := buildStatefulSet(mls, Config{})

	if got := sts.Spec.Template.Spec.SchedulerName; got != axisml.SchedulerName {
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
		if got := sts.Spec.Template.Labels[k]; got != want {
			t.Errorf("pod label %q = %q; want %q", k, got, want)
		}
	}

	// Selector must NOT carry the quota / tenant labels.
	if _, ok := sts.Spec.Selector.MatchLabels[axisml.LabelKoordQuotaName]; ok {
		t.Error("selector unexpectedly contains koord quota label")
	}
}

func TestBuildStatefulSet_InjectsModelEnvVar(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{})
	c := sts.Spec.Template.Spec.Containers[0]
	if !envHasValue(c.Env, modelEnvVarName, "model://dummy:v1") {
		t.Errorf("env var %s not injected with expected value; got %v", modelEnvVarName, c.Env)
	}
}

func TestBuildStatefulSet_InjectsReplicaIndexEnvVar(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{})
	c := sts.Spec.Template.Spec.Containers[0]
	for _, e := range c.Env {
		if e.Name == replicaIndexEnvVarName {
			if e.ValueFrom == nil || e.ValueFrom.FieldRef == nil {
				t.Fatalf("%s missing fieldRef downward API: %+v", replicaIndexEnvVarName, e)
			}
			if e.ValueFrom.FieldRef.FieldPath != replicaIndexFieldPath {
				t.Errorf("fieldPath = %q; want %q",
					e.ValueFrom.FieldRef.FieldPath, replicaIndexFieldPath)
			}
			return
		}
	}
	t.Errorf("env var %s not injected; got %v", replicaIndexEnvVarName, c.Env)
}

func TestBuildStatefulSet_DefaultsServiceName(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{})
	if sts.Spec.ServiceName != "smoke" {
		t.Errorf("ServiceName = %q; want %q (defaulted to MLService name)",
			sts.Spec.ServiceName, "smoke")
	}
}

func TestBuildStatefulSet_HonorsServiceName(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{ServiceName: "custom-headless"})
	if sts.Spec.ServiceName != "custom-headless" {
		t.Errorf("ServiceName = %q; want %q", sts.Spec.ServiceName, "custom-headless")
	}
}

func TestBuildStatefulSet_DefaultsPodManagementPolicy(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{})
	if sts.Spec.PodManagementPolicy != appsv1.OrderedReadyPodManagement {
		t.Errorf("podManagementPolicy = %q; want %q",
			sts.Spec.PodManagementPolicy, appsv1.OrderedReadyPodManagement)
	}
}

func TestBuildStatefulSet_HonorsParallelPodManagement(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := buildStatefulSet(mls, Config{PodManagementPolicy: appsv1.ParallelPodManagement})
	if sts.Spec.PodManagementPolicy != appsv1.ParallelPodManagement {
		t.Errorf("podManagementPolicy = %q; want Parallel", sts.Spec.PodManagementPolicy)
	}
}

func TestBuildHeadlessService_ClusterIPNone(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	svc := buildHeadlessService(mls, Config{})
	if svc.Spec.ClusterIP != corev1.ClusterIPNone {
		t.Errorf("ClusterIP = %q; want %q (headless)", svc.Spec.ClusterIP, corev1.ClusterIPNone)
	}
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 service port; got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].TargetPort.IntValue() != 8080 {
		t.Errorf("targetPort = %v; want 8080", svc.Spec.Ports[0].TargetPort)
	}
}

func TestBuildHeadlessService_HonorsServiceName(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	svc := buildHeadlessService(mls, Config{ServiceName: "custom-headless"})
	if svc.Name != "custom-headless" {
		t.Errorf("Service.Name = %q; want %q", svc.Name, "custom-headless")
	}
}

func TestMapStatus_PhasePending_NoStatefulSet(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	upd := mapStatus(handler.Snapshot{Service: mls})
	if upd.Phase != axisml.PhasePending {
		t.Errorf("phase = %s; want Pending", upd.Phase)
	}
}

func TestMapStatus_PhaseReady_WhenStatefulSetReady(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{sts, svc},
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

func TestMapStatus_PhaseDegraded_PartialReady(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	mls.Spec.Roles[0].Replicas = 3
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{sts},
	})
	if upd.Phase != axisml.PhaseDegraded {
		t.Errorf("phase = %s; want Degraded", upd.Phase)
	}
}

func TestMapStatus_PhasePending_RollingOut(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 0},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{sts},
	})
	if upd.Phase != axisml.PhasePending {
		t.Errorf("phase = %s; want Pending (rolling out)", upd.Phase)
	}
}

func TestMapStatus_RouteEnabled_DegradedWhenNotAccepted(t *testing.T) {
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *minimalSpec(),
	}
	mls.Spec.Route = &axisml.Route{Enabled: true, Hostname: "demo.example.com"}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	route := &gwapiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{sts, svc, route},
	})
	if upd.Phase != axisml.PhaseDegraded {
		t.Errorf("phase = %s; want Degraded (StatefulSet ready but route not Accepted)", upd.Phase)
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
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
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
		Children: []client.Object{sts, svc, route},
	})
	if upd.Phase != axisml.PhaseReady {
		t.Errorf("phase = %s; want Ready", upd.Phase)
	}
	if upd.Endpoint != "https://demo.example.com/predict" {
		t.Errorf("endpoint = %q; want external URL", upd.Endpoint)
	}
}

func TestMapStatus_HonorsCustomServiceName(t *testing.T) {
	spec := minimalSpec()
	setConfig(spec, `{"serviceName":"custom-headless"}`)
	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Spec:       *spec,
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "smoke", Namespace: "tenant-demo"},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "custom-headless", Namespace: "tenant-demo"},
	}
	upd := mapStatus(handler.Snapshot{
		Service:  mls,
		Children: []client.Object{sts, svc},
	})
	if upd.Endpoint != "custom-headless.tenant-demo.svc.cluster.local:8080" {
		t.Errorf("endpoint = %q; want headless-name-based DNS", upd.Endpoint)
	}
}

func envHasValue(env []corev1.EnvVar, name, value string) bool {
	for _, e := range env {
		if e.Name == name && e.Value == value {
			return true
		}
	}
	return false
}

func containsSubstring(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}
