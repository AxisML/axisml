//go:build integration

package integration_test

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

	axisml "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

// TestMLService_NativeStatefulSet_HappyPath verifies the dispatcher creates
// a StatefulSet + headless Service (clusterIP=None) and converges MLService
// to Ready once the StatefulSet's status reports replicas ready. envtest has
// no kubelet, so the rollout-completing status is simulated via patch.
func TestMLService_NativeStatefulSet_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc-sts"
		svcName = "predictor-sts"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-predictor-sts",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "statefulset"},
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

	// Dispatcher creates the StatefulSet.
	var sts appsv1.StatefulSet
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedStatefulSet(ctx, c, ns, svcName, &sts)
	})
	require.Equal(t, svcName, sts.Spec.ServiceName,
		"StatefulSet.spec.serviceName must default to the MLService name")

	// And the headless Service.
	var k8sSvc corev1.Service
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedService(ctx, c, ns, svcName, &k8sSvc)
	})
	require.Equal(t, corev1.ClusterIPNone, k8sSvc.Spec.ClusterIP,
		"Service must be headless (clusterIP=None)")

	markStatefulSetReady(t, ctx, c, &sts, 1)

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

// TestMLService_NativeStatefulSet_ScaleAndImmutability covers the two
// dispatcher-level mutation rules for statefulset workloads: replicas is
// mutable and propagates to StatefulSet.spec.replicas; mutating any other
// field surfaces phase=Failed with an immutable-message.
func TestMLService_NativeStatefulSet_ScaleAndImmutability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		ns      = "envt-mlsvc-sts-scale"
		svcName = "scalable-sts"
	)
	require.NoError(t, c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}))
	t.Cleanup(func() { cleanupNamespace(t, c, ns) })

	mls := &axisml.MLService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      svcName,
			Labels: map[string]string{
				axisml.LabelServiceID: "uuid-scalable-sts",
				axisml.LabelTenant:    "acme",
				axisml.LabelQuota:     "axisml-acme-default-default",
			},
		},
		Spec: axisml.MLServiceSpec{
			Backend:    axisml.Backend{Name: "native", Engine: "statefulset"},
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

	// Wait for the StatefulSet, then patch its status to Ready so the
	// dispatcher stamps the immutable hash baseline.
	var sts appsv1.StatefulSet
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		return findOwnedStatefulSet(ctx, c, ns, svcName, &sts)
	})
	markStatefulSetReady(t, ctx, c, &sts, 1)

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
		var got appsv1.StatefulSet
		if err := findOwnedStatefulSet(ctx, c, ns, svcName, &got); err != nil {
			return err
		}
		if got.Spec.Replicas == nil || *got.Spec.Replicas != 3 {
			return fmt.Errorf("StatefulSet.spec.replicas not 3")
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

// markStatefulSetReady writes a status block that satisfies derivePhase's
// "all replicas ready" branch. envtest has no kubelet, so the StatefulSet
// controller never reports its own progress — tests stamp this directly.
func markStatefulSetReady(t *testing.T, ctx context.Context, c client.Client, sts *appsv1.StatefulSet, replicas int32) {
	t.Helper()
	patch := client.MergeFrom(sts.DeepCopy())
	sts.Status.ObservedGeneration = sts.Generation
	sts.Status.Replicas = replicas
	sts.Status.ReadyReplicas = replicas
	sts.Status.AvailableReplicas = replicas
	sts.Status.CurrentReplicas = replicas
	sts.Status.UpdatedReplicas = replicas
	sts.Status.CurrentRevision = "sts-revision-1"
	sts.Status.UpdateRevision = "sts-revision-1"
	require.NoError(t, c.Status().Patch(ctx, sts, patch))
}

// findOwnedStatefulSet locates the StatefulSet whose ownerRef points at the
// MLService named svcName. Mirrors findOwnedDeployment / findOwnedService.
func findOwnedStatefulSet(ctx context.Context, c client.Client, ns, svcName string, out *appsv1.StatefulSet) error {
	var list appsv1.StatefulSetList
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
	return fmt.Errorf("no StatefulSet owned by MLService %s/%s", ns, svcName)
}
