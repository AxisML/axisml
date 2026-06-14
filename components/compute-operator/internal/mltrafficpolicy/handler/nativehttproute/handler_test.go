package nativehttproute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
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

func TestValidate_JWTAuthEmitsDeferralWarning(t *testing.T) {
	h := &Handler{}
	spec := weightedSpec()
	spec.Endpoint.Auth = &mltp.EndpointAuth{Type: mltp.EndpointAuthJWT}
	v := h.Validate(spec)
	assert.True(t, v.OK())
	assert.NotEmpty(t, v.Warnings)
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
