//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

var nameSeq int64

// mustQty parses a resource quantity string, panicking on error (test-only).
func mustQty(s string) resource.Quantity { return resource.MustParse(s) }

// uniqueName returns a short, DNS-safe, run-unique name with the given prefix.
func uniqueName(prefix string) string {
	n := atomic.AddInt64(&nameSeq, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix()%100000, n)
}

// buildTenant builds a Tenant CR with one quota on the default pool. cpu/mem are
// resource quantity strings (e.g. "1", "2Gi").
func buildTenant(name, namespace, pool, cpu, mem string) *tenantv1.Tenant {
	return &tenantv1.Tenant{
		// The operator requires a non-empty tenant-id label (its orphan-detection
		// anchor); compute-service stamps a UUID here. The name is unique enough
		// for tests.
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{tenantv1.LabelTenantID: name},
		},
		Spec: tenantv1.TenantSpec{
			Namespace: tenantv1.NamespaceSpec{Name: namespace},
			Quotas: []tenantv1.QuotaSpec{{
				Pool: pool,
				Name: "default",
				Max: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(cpu),
					corev1.ResourceMemory: resource.MustParse(mem),
				},
			}},
		},
	}
}

// createTenantCR applies a Tenant directly via the K8s API (isolating the
// tenant-operator from compute-service) and registers cleanup that deletes the
// Tenant and waits for its namespace to disappear.
func createTenantCR(t *testing.T, ctx context.Context, ten *tenantv1.Tenant) {
	t.Helper()
	require := func(err error, msg string) {
		if err != nil {
			t.Fatalf("%s: %v", msg, err)
		}
	}
	require(h.k8s.Create(ctx, ten), "create Tenant")
	t.Cleanup(func() {
		bg := context.Background()
		_ = h.k8s.Delete(bg, ten.DeepCopy())
		// The operator intentionally never deletes the tenant namespace (design
		// §6.1: "never delete, no ownerReference"), so the test runner (admin
		// kubeconfig) removes it to keep re-runs clean. Best-effort, no wait.
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ten.Spec.Namespace.Name}}
		_ = h.k8s.Delete(bg, ns)
	})
}

// getTenantCR fetches the current Tenant object.
func getTenantCR(ctx context.Context, name string) (*tenantv1.Tenant, error) {
	var ten tenantv1.Tenant
	if err := h.k8s.Get(ctx, client.ObjectKey{Name: name}, &ten); err != nil {
		return nil, err
	}
	return &ten, nil
}

// schedulablePod builds a minimal pod that schedules under a koord ElasticQuota.
// It carries the quota label and koord-scheduler, requesting `cpu` CPU.
func schedulablePod(ns, name, quotaName, cpu string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    map[string]string{"quota.scheduling.koordinator.sh/name": quotaName},
		},
		Spec: corev1.PodSpec{
			SchedulerName: "koord-scheduler",
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:            "pause",
				Image:           h.cfg.MLRunImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"sh", "-c", "sleep 600"},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)},
				},
			}},
		},
	}
}

// podScheduled reports whether the pod has been assigned to a node.
func podScheduled(ctx context.Context, ns, name string) (bool, error) {
	var p corev1.Pod
	if err := h.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, &p); err != nil {
		return false, err
	}
	return p.Spec.NodeName != "", nil
}
