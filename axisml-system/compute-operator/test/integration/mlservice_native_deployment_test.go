//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLService_NativeDeployment_HappyPath drives the (native, deployment)
// handler end-to-end:
//
//  1. Create a Namespace and an MLService{native, deployment} CR with one
//     predictor role running nginx.
//  2. Assert the dispatcher creates a Deployment + Service.
//  3. Patch the Deployment status to simulate the rollout completing
//     (envtest has no replicaset/scheduler/kubelet).
//  4. Assert MLService.Status.Phase converges to Ready.
func TestMLService_NativeDeployment_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc"
		svcName = "predictor"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	svc := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-predictor",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "deployment"},
			Scheduling: axisml.Scheduling{Quota: "axisml-acme-default"},
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
	require.NoError(t, c.Create(ctx, svc))

	// Dispatcher creates the Deployment.
	var dep appsv1.Deployment
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedDeployment(ctx, c, ns, svcName, &dep)
	})

	// And the Service.
	var k8sSvc corev1.Service
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedService(ctx, c, ns, svcName, &k8sSvc)
	})

	// Simulate the rollout completing.
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

// findOwnedDeployment locates the Deployment whose ownerRef points at the
// MLService named svcName. The handler picks the actual Deployment name
// (we don't hardcode it); ownerRef is the stable anchor.
func findOwnedDeployment(ctx context.Context, c client.Client, ns, svcName string, out *appsv1.Deployment) error {
	var list appsv1.DeploymentList
	if err := c.List(ctx, &list, client.InNamespace(ns)); err != nil {
		return err
	}
	for i := range list.Items {
		for _, o := range list.Items[i].OwnerReferences {
			if o.Kind == "MLService" && o.Name == svcName {
				*out = list.Items[i]
				return nil
			}
		}
	}
	return fmt.Errorf("no Deployment owned by MLService %s/%s", ns, svcName)
}

func findOwnedService(ctx context.Context, c client.Client, ns, svcName string, out *corev1.Service) error {
	var list corev1.ServiceList
	if err := c.List(ctx, &list, client.InNamespace(ns)); err != nil {
		return err
	}
	for i := range list.Items {
		for _, o := range list.Items[i].OwnerReferences {
			if o.Kind == "MLService" && o.Name == svcName {
				*out = list.Items[i]
				return nil
			}
		}
	}
	return fmt.Errorf("no Service owned by MLService %s/%s", ns, svcName)
}
