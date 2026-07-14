package nativehttproute

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
	trafficpolicyhandler "github.com/axisml/axisml/axisml-system/compute-operator/internal/mltrafficpolicy/handler"
)

func weightedSpec() *mltp.MLTrafficPolicySpec {
	return &mltp.MLTrafficPolicySpec{
		Backend:  mltp.Backend{Name: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute},
		Mode:     mltp.TrafficModeWeighted,
		Endpoint: mltp.Endpoint{Path: "/services/acme/chat/", Hostname: "chat.example.com"},
		Backends: []mltp.BackendMember{
			{ServiceName: "chat-v1", Weight: 90},
			{ServiceName: "chat-v2", Weight: 10},
		},
	}
}

func TestValidate_WeightedHappyPath(t *testing.T) {
	h := &Handler{}
	v := h.Validate(weightedSpec())
	assert.True(t, v.OK(), "errors: %v", v.Errors)
}

func TestValidate_WeightedNeedsTwoBackends(t *testing.T) {
	h := &Handler{}
	spec := weightedSpec()
	spec.Backends = spec.Backends[:1]
	v := h.Validate(spec)
	assert.False(t, v.OK())
}

func TestValidate_CanaryRequiresStableAndCanaryRoles(t *testing.T) {
	h := &Handler{}
	spec := &mltp.MLTrafficPolicySpec{
		Backend:  mltp.Backend{Name: mltp.BackendKindNative, Engine: mltp.EngineHTTPRoute},
		Mode:     mltp.TrafficModeCanary,
		Endpoint: mltp.Endpoint{Path: "/services/acme/chat/"},
		Backends: []mltp.BackendMember{
			{ServiceName: "chat-v1", Role: mltp.RoleStable, Weight: 90},
			{ServiceName: "chat-v2", Role: mltp.RoleCanary, Weight: 10},
		},
	}
	require.True(t, h.Validate(spec).OK())

	// both stable → invalid
	spec.Backends[1].Role = mltp.RoleStable
	assert.False(t, h.Validate(spec).OK())
}

func TestValidate_RejectsDuplicateAndBadWeight(t *testing.T) {
	h := &Handler{}
	spec := weightedSpec()
	spec.Backends[1].ServiceName = "chat-v1" // duplicate
	spec.Backends[1].Weight = 250            // out of range
	v := h.Validate(spec)
	assert.False(t, v.OK())
	assert.GreaterOrEqual(t, len(v.Errors), 2)
}

func TestValidate_UnknownMode(t *testing.T) {
	h := &Handler{}
	spec := weightedSpec()
	spec.Mode = "rollercoaster"
	assert.False(t, h.Validate(spec).OK())
}

func TestValidate_JWTAuthRejectedUntilWired(t *testing.T) {
	// Fail closed: SecurityPolicy is not wired yet, so auth=jwt must be
	// rejected rather than silently programming an unauthenticated route.
	h := &Handler{}
	spec := weightedSpec()
	spec.Endpoint.Auth = &mltp.EndpointAuth{Type: mltp.EndpointAuthJWT}
	assert.False(t, h.Validate(spec).OK())

	// auth.type=none stays valid.
	spec.Endpoint.Auth = &mltp.EndpointAuth{Type: mltp.EndpointAuthNone}
	assert.True(t, h.Validate(spec).OK())
}

func reconcileScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, gwapiv1.Install(s))
	require.NoError(t, mltp.AddToScheme(s))
	return s
}

func svc(ns, name string, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Port: port}}},
	}
}

func weightedPolicy() *mltp.MLTrafficPolicy {
	p := &mltp.MLTrafficPolicy{Spec: *weightedSpec()}
	p.Name = "chat"
	p.Namespace = "acme"
	p.Labels = map[string]string{mltp.LabelTrafficPolicyID: "uuid-chat", mltp.LabelTenant: "acme"}
	return p
}

// A weighted policy whose member services do not all resolve must NOT program a
// partial route (Gateway API would renormalise weights onto the survivors and
// shift traffic). Reconcile errors to requeue and no HTTPRoute is created.
func TestReconcile_PartialMembers_HoldsRouteAndErrors(t *testing.T) {
	s := reconcileScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).
		WithObjects(svc("acme", "chat-v1", 8080)). // chat-v2 missing
		Build()
	h := &Handler{client: c}

	_, err := h.Reconcile(context.Background(), weightedPolicy())
	require.Error(t, err, "expected reconcile to hold and requeue on partial member set")

	routes := &gwapiv1.HTTPRouteList{}
	require.NoError(t, c.List(context.Background(), routes, client.InNamespace("acme")))
	assert.Empty(t, routes.Items, "no HTTPRoute should be programmed from a partial member set")
}

// Once every member resolves, the weighted route is programmed with all members.
func TestReconcile_AllMembers_ProgramsWeightedRoute(t *testing.T) {
	s := reconcileScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).
		WithObjects(svc("acme", "chat-v1", 8080), svc("acme", "chat-v2", 8080)).
		Build()
	h := &Handler{client: c}

	_, err := h.Reconcile(context.Background(), weightedPolicy())
	require.NoError(t, err)

	route := &gwapiv1.HTTPRoute{}
	require.NoError(t, c.Get(context.Background(), types.NamespacedName{Namespace: "acme", Name: "chat"}, route))
	require.Len(t, route.Spec.Rules, 1)
	assert.Len(t, route.Spec.Rules[0].BackendRefs, 2)
}

func TestBuildHTTPRoute_WeightsPortsHostPath(t *testing.T) {
	p := &mltp.MLTrafficPolicy{
		Spec: *weightedSpec(),
	}
	p.Name = "chat"
	p.Namespace = "acme"
	p.Labels = map[string]string{mltp.LabelTrafficPolicyID: "uuid-chat", mltp.LabelTenant: "acme"}

	refs := []gwapiv1.HTTPBackendRef{
		backendRef("chat-v1", 8080, 90),
		backendRef("chat-v2", 8080, 10),
	}
	route := buildHTTPRoute(p, refs)

	// label so the dispatcher can list it as an owned child
	assert.Equal(t, "uuid-chat", route.Labels[mltp.LabelTrafficPolicyID])

	require.Len(t, route.Spec.ParentRefs, 1)
	assert.Equal(t, gwapiv1.ObjectName(GatewayParentName), route.Spec.ParentRefs[0].Name)
	require.NotNil(t, route.Spec.ParentRefs[0].Namespace)
	assert.Equal(t, gwapiv1.Namespace(GatewayParentNamespace), *route.Spec.ParentRefs[0].Namespace)

	require.Len(t, route.Spec.Hostnames, 1)
	assert.Equal(t, gwapiv1.Hostname("chat.example.com"), route.Spec.Hostnames[0])

	require.Len(t, route.Spec.Rules, 1)
	rule := route.Spec.Rules[0]
	require.Len(t, rule.Matches, 1)
	require.NotNil(t, rule.Matches[0].Path.Value)
	assert.Equal(t, "/services/acme/chat/", *rule.Matches[0].Path.Value)

	require.Len(t, rule.BackendRefs, 2)
	require.NotNil(t, rule.BackendRefs[0].Weight)
	assert.Equal(t, int32(90), *rule.BackendRefs[0].Weight)
	require.NotNil(t, rule.BackendRefs[1].Weight)
	assert.Equal(t, int32(10), *rule.BackendRefs[1].Weight)
	require.NotNil(t, rule.BackendRefs[0].Port)
	assert.Equal(t, gwapiv1.PortNumber(8080), *rule.BackendRefs[0].Port)
}

func TestMapStatusFindsTenantPrefixedRoute(t *testing.T) {
	p := weightedPolicy()
	workloadname.Annotate(p, "acme", p.Name, true)
	route := &gwapiv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{
		Namespace: p.Namespace,
		Name:      workloadname.Workload(p),
	}}
	route.Status.Parents = []gwapiv1.RouteParentStatus{{Conditions: []metav1.Condition{
		{Type: string(gwapiv1.RouteConditionAccepted), Status: metav1.ConditionTrue},
		{Type: string(gwapiv1.RouteConditionResolvedRefs), Status: metav1.ConditionTrue},
	}}}

	got := mapStatus(trafficpolicyhandler.Snapshot{Policy: p, Children: []client.Object{route}})
	assert.Equal(t, mltp.PhaseReady, got.Phase)
}
