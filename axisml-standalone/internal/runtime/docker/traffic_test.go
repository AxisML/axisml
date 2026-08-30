package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"
)

func TestGatewayTrafficResourcesAndReadback(t *testing.T) {
	r := New(nil, Config{GatewayConfigDir: t.TempDir()}, logr.Discard())
	policy := &mltp.MLTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant-a", Name: "rollout"},
		Spec: mltp.MLTrafficPolicySpec{
			Endpoint: mltp.Endpoint{Hostname: "model.example.com", Path: "/predict"},
		},
	}
	backends := []trafficBackend{
		{
			serviceName:  "stable",
			resourceName: r.trafficMemberResourceName("tenant-a", "rollout", "stable"),
			endpoints:    []gatewayEndpoint{gatewayContainerEndpoint("stable-predictor-0", 8080)},
			weight:       90,
		},
		{
			serviceName:  "canary",
			resourceName: r.trafficMemberResourceName("tenant-a", "rollout", "canary"),
			endpoints:    []gatewayEndpoint{gatewayContainerEndpoint("canary-predictor-0", 8080)},
			weight:       10,
		},
	}

	b, err := marshalGatewayResources(r.gatewayTrafficResources(policy, backends)...)
	require.NoError(t, err)
	content := string(b)
	assert.Equal(t, 2, strings.Count(content, "apiVersion: gateway.envoyproxy.io/v1alpha1\n"))
	assert.Contains(t, content, "hostname: stable-predictor-0.axisml.local")
	assert.Contains(t, content, "hostname: canary-predictor-0.axisml.local")
	assert.Contains(t, content, "kind: HTTPRoute")
	assert.Contains(t, content, "weight: 90")
	assert.Contains(t, content, "weight: 10")
	assert.Contains(t, content, "value: /predict")

	path := filepath.Join(t.TempDir(), "traffic-policy.yaml")
	require.NoError(t, os.WriteFile(path, b, 0o644))
	members, err := r.readTrafficMembers(path)
	require.NoError(t, err)
	require.Len(t, members, 2)
	assert.Equal(t, trafficMember{
		service: "stable", weight: 90, path: "/predict", host: "model.example.com",
	}, members[0])
	assert.Equal(t, trafficMember{
		service: "canary", weight: 10, path: "/predict", host: "model.example.com",
	}, members[1])
	assert.Equal(t, "http://model.example.com/predict", r.trafficEndpoint(members))
}

func TestTrafficMemberResourceNameIncludesPolicy(t *testing.T) {
	r := New(nil, Config{}, logr.Discard())
	one := r.trafficMemberResourceName("tenant-a", "policy-one", "model")
	two := r.trafficMemberResourceName("tenant-a", "policy-two", "model")
	assert.NotEqual(t, one, two)
}

func TestGatewayBackendWithoutReplicasUsesUnavailableEndpoint(t *testing.T) {
	b, err := marshalGatewayResources(gatewayBackend("model", "model", nil))
	require.NoError(t, err)
	content := string(b)
	assert.Contains(t, content, "address: 127.0.0.1")
	assert.Contains(t, content, "port: 1")
	assert.NotContains(t, content, "fqdn:")
}
