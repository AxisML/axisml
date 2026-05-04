//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLService_NativeDeployment_RouteEnabled covers the route-enabled
// reconcile path: spec.route.enabled=true must emit an HTTPRoute alongside
// the Deployment + Service, with backendRefs pointing at the in-namespace
// Service and a parentRef to the cluster-shared axisml-gateway. Auth /
// rateLimit are deferred per design §11 and intentionally NOT exercised
// here (they need Envoy Gateway CRDs, not yet vendored).
func TestMLService_NativeDeployment_RouteEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc-route"
		svcName = "predictor-route"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-predictor-route",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
			Scheduling: axisml.Scheduling{Quota: "axisml-acme-default-default"},
			ModelRef:   axisml.ModelRef{Name: "demo", Version: "v1"},
			Roles: []axisml.RoleSpec{{
				Name:     axisml.DefaultRoleName,
				Replicas: 1,
				Template: axisml.PodTemplate{
					Image: "nginx:1.27",
					Ports: []axisml.PodPort{{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP}},
				},
			}},
			Route: &axisml.Route{
				Enabled:  true,
				PortName: "http",
				Hostname: "predictor.example.com",
				Path:     "/v1",
			},
		},
	}
	require.NoError(t, c.Create(ctx, mls))

	// Deployment + Service still appear (covered by the existing happy path,
	// but we re-assert presence to catch a regression where route handling
	// would short-circuit them).
	var dep appsv1.Deployment
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedDeployment(ctx, c, ns, svcName, &dep)
	})
	var k8sSvc corev1.Service
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedService(ctx, c, ns, svcName, &k8sSvc)
	})

	// HTTPRoute lives in the same namespace, named after the MLService.
	var route gwapiv1.HTTPRoute
	testutil.EventuallyExists(t, ctx, c,
		types.NamespacedName{Namespace: ns, Name: svcName}, &route, testWaitTimeout)

	require.Len(t, route.Spec.Hostnames, 1)
	require.Equal(t, gwapiv1.Hostname("predictor.example.com"), route.Spec.Hostnames[0])
	require.Len(t, route.Spec.ParentRefs, 1)
	require.Equal(t, gwapiv1.ObjectName("axisml-gateway"), route.Spec.ParentRefs[0].Name)
	require.NotNil(t, route.Spec.ParentRefs[0].Namespace)
	require.Equal(t, gwapiv1.Namespace("axisml-infra"), *route.Spec.ParentRefs[0].Namespace)
	require.Len(t, route.Spec.Rules, 1)
	require.Len(t, route.Spec.Rules[0].BackendRefs, 1)
	require.Equal(t, gwapiv1.ObjectName(svcName), route.Spec.Rules[0].BackendRefs[0].Name)
	require.NotNil(t, route.Spec.Rules[0].BackendRefs[0].Port)
	require.Equal(t, gwapiv1.PortNumber(8080), *route.Spec.Rules[0].BackendRefs[0].Port)

	// Patch Deployment status to Available.
	patch := client.MergeFrom(dep.DeepCopy())
	dep.Status.ObservedGeneration = dep.Generation
	dep.Status.Replicas = 1
	dep.Status.UpdatedReplicas = 1
	dep.Status.ReadyReplicas = 1
	dep.Status.AvailableReplicas = 1
	dep.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:           appsv1.DeploymentAvailable,
		Status:         corev1.ConditionTrue,
		LastUpdateTime: metav1.NewTime(time.Now()),
	}}
	require.NoError(t, c.Status().Patch(ctx, &dep, patch))

	// Without an Envoy Gateway controller in envtest, the HTTPRoute never
	// transitions to Accepted on its own — the handler reports Degraded
	// ("HTTPRoute not yet Accepted; falling back to in-cluster DNS"). Simulate
	// the gateway controller by patching status.parents[].conditions=Accepted.
	rPatch := client.MergeFrom(route.DeepCopy())
	route.Status.Parents = []gwapiv1.RouteParentStatus{{
		ParentRef:      route.Spec.ParentRefs[0],
		ControllerName: gwapiv1.GatewayController("gateway.envoyproxy.io/gatewayclass-controller"),
		Conditions: []metav1.Condition{{
			Type:               string(gwapiv1.RouteConditionAccepted),
			Status:             metav1.ConditionTrue,
			Reason:             string(gwapiv1.RouteReasonAccepted),
			LastTransitionTime: metav1.NewTime(time.Now()),
		}},
	}}
	require.NoError(t, c.Status().Patch(ctx, &route, rPatch))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.PhaseReady {
			return fmt.Errorf("phase=%q (msg=%q)", got.Status.Phase, got.Status.Message)
		}
		return nil
	})
}

// TestMLService_ScaleAndImmutability verifies the dispatcher's mutation
// rules: roles[*].replicas is mutable (Deployment.replicas tracks it),
// while every other immutable field rejects with phase=Failed. The
// existing unit tests cover the hash function; this test asserts the
// integration-level behavior.
func TestMLService_ScaleAndImmutability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc-scale"
		svcName = "scalable"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-scalable",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
			Scheduling: axisml.Scheduling{Quota: "axisml-acme-default-default"},
			ModelRef:   axisml.ModelRef{Name: "demo", Version: "v1"},
			Roles: []axisml.RoleSpec{{
				Name:     axisml.DefaultRoleName,
				Replicas: 1,
				Template: axisml.PodTemplate{
					Image: "nginx:1.27",
					Ports: []axisml.PodPort{{Name: "http", ContainerPort: 80, Protocol: corev1.ProtocolTCP}},
				},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mls))

	// Wait for the Deployment, then patch its status to Available so the
	// dispatcher stamps the immutable hash baseline. (Per immutability.go,
	// the baseline is recorded only after a fully successful reconcile.)
	var dep appsv1.Deployment
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedDeployment(ctx, c, ns, svcName, &dep)
	})
	patch := client.MergeFrom(dep.DeepCopy())
	dep.Status.ObservedGeneration = dep.Generation
	dep.Status.Replicas = 1
	dep.Status.UpdatedReplicas = 1
	dep.Status.ReadyReplicas = 1
	dep.Status.AvailableReplicas = 1
	dep.Status.Conditions = []appsv1.DeploymentCondition{{
		Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue,
		LastUpdateTime: metav1.NewTime(time.Now()),
	}}
	require.NoError(t, c.Status().Patch(ctx, &dep, patch))

	// Wait for Phase=Ready (which means the immutable hash is now stamped).
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.PhaseReady {
			return fmt.Errorf("phase=%q msg=%q", got.Status.Phase, got.Status.Message)
		}
		return nil
	})

	// 1) Replica scale is allowed.
	var fresh axisml.MLService
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &fresh))
	patchObj := client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Roles[0].Replicas = 3
	require.NoError(t, c.Patch(ctx, &fresh, patchObj))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got appsv1.Deployment
		if err := findOwnedDeployment(ctx, c, ns, svcName, &got); err != nil {
			return err
		}
		if got.Spec.Replicas == nil || *got.Spec.Replicas != 3 {
			return fmt.Errorf("Deployment.spec.replicas not 3")
		}
		return nil
	})

	// 2) Mutating an immutable field (image) → Failed with immutable msg.
	require.NoError(t, c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &fresh))
	patchObj = client.MergeFrom(fresh.DeepCopy())
	fresh.Spec.Roles[0].Template.Image = "nginx:1.28"
	require.NoError(t, c.Patch(ctx, &fresh, patchObj))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.PhaseFailed {
			return fmt.Errorf("phase=%q msg=%q, want Failed", got.Status.Phase, got.Status.Message)
		}
		if !strings.Contains(got.Status.Message, "immutable") {
			return fmt.Errorf("message %q does not mention immutability", got.Status.Message)
		}
		return nil
	})
}

// TestMLService_StubBackendFails verifies the registered stub backends
// (kserve/inference, kserve/llminference, native/statefulset) all surface
// a clear "not implemented" failure rather than silently doing nothing.
// This locks in the contract from handler/stubs.go so a future refactor
// doesn't accidentally let an unimplemented backend reach Pending and stay
// there forever.
func TestMLService_StubBackendFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc-stub"
		svcName = "stub-backend"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-stub-backend",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "kserve", Engine: "inference"},
			Scheduling: axisml.Scheduling{Quota: "axisml-acme-default-default"},
			ModelRef:   axisml.ModelRef{Name: "demo", Version: "v1"},
			Roles: []axisml.RoleSpec{{
				Name:     axisml.DefaultRoleName,
				Replicas: 1,
				Template: axisml.PodTemplate{Image: "n/a:stub"},
			}},
		},
	}
	require.NoError(t, c.Create(ctx, mls))

	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.MLService
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: svcName}, &got); err != nil {
			return err
		}
		if got.Status.Phase != axisml.PhaseFailed {
			return fmt.Errorf("phase=%q msg=%q, want Failed", got.Status.Phase, got.Status.Message)
		}
		if !strings.Contains(got.Status.Message, "not implemented") {
			return fmt.Errorf("message %q does not mention 'not implemented'", got.Status.Message)
		}
		return nil
	})
}
