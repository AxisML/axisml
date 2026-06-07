//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// podReady reports whether a pod is Running with its Ready condition true, or
// has already Succeeded (for run-to-completion jobs).
func podReady(p *corev1.Pod) bool {
	if p.Status.Phase == corev1.PodSucceeded {
		return true
	}
	if p.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

// readyPodNames returns the names of ready pods in ns whose name contains sub.
func readyPodNames(ctx context.Context, ns, sub string) ([]string, error) {
	var pods corev1.PodList
	if err := h.k8s.List(ctx, &pods, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	var out []string
	for i := range pods.Items {
		p := &pods.Items[i]
		if sub != "" && !strings.Contains(p.Name, sub) {
			continue
		}
		if podReady(p) {
			out = append(out, p.Name)
		}
	}
	return out, nil
}

// requireReadyPod fails unless at least one ready pod in ns matches sub.
func requireReadyPod(t *testing.T, ctx context.Context, ns, sub string) {
	t.Helper()
	eventually(t, h.cfg.PodReadyTimeout, func() error {
		names, err := readyPodNames(ctx, ns, sub)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return fmt.Errorf("no ready pod matching %q in %s", sub, ns)
		}
		return nil
	})
}

// logReadyPod logs (without failing) whether a matching ready pod exists — used
// for optional components like the GPU operator.
func logReadyPod(t *testing.T, ctx context.Context, ns, sub string) {
	t.Helper()
	names, err := readyPodNames(ctx, ns, sub)
	if err != nil || len(names) == 0 {
		t.Logf("optional component %q: no ready pod (err=%v)", sub, err)
		return
	}
	t.Logf("optional component %q: ready (%v)", sub, names)
}

// crdEstablished returns nil once the named CRD reports Established=True.
func crdEstablished(ctx context.Context, name string) error {
	crd := &unstructured.Unstructured{}
	crd.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	if err := h.k8s.Get(ctx, client.ObjectKey{Name: name}, crd); err != nil {
		return err
	}
	conds, found, err := unstructured.NestedSlice(crd.Object, "status", "conditions")
	if err != nil || !found {
		return fmt.Errorf("CRD %s has no status.conditions", name)
	}
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == "Established" && m["status"] == "True" {
			return nil
		}
	}
	return fmt.Errorf("CRD %s not Established", name)
}

// countRoles returns the number of RBAC Roles in a namespace (the tenant
// operator creates at least one; the namespace ships with none by default).
func countRoles(ctx context.Context, ns string) (int, error) {
	var roles rbacv1.RoleList
	if err := h.k8s.List(ctx, &roles, client.InNamespace(ns)); err != nil {
		return 0, err
	}
	return len(roles.Items), nil
}

// assertErr is a fmt.Errorf shorthand for use inside polling closures.
func assertErr(format string, args ...any) error { return fmt.Errorf(format, args...) }

// newQuotaList returns an empty typed-as-unstructured ElasticQuota list.
func newQuotaList() *unstructured.UnstructuredList {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "scheduling.sigs.k8s.io",
		Version: "v1alpha1",
		Kind:    "ElasticQuotaList",
	})
	return list
}

// elasticQuotaNames lists ElasticQuota names in a namespace (no *testing.T, so
// it is usable inside polling closures).
func elasticQuotaNames(ctx context.Context, ns string) ([]string, error) {
	list := newQuotaList()
	if err := h.k8s.List(ctx, list, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for i := range list.Items {
		names = append(names, list.Items[i].GetName())
	}
	return names, nil
}

// quotaMax reads .spec.max of a named ElasticQuota as a string map.
func quotaMax(ctx context.Context, ns, name string) (map[string]string, error) {
	eq := quotaObj()
	if err := h.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, eq); err != nil {
		return nil, err
	}
	raw, found, err := unstructured.NestedStringMap(eq.Object, "spec", "max")
	if err != nil || !found {
		return nil, fmt.Errorf("ElasticQuota %s/%s has no spec.max", ns, name)
	}
	return raw, nil
}

// newBatchJob returns an empty unstructured batch/v1 Job.
func newBatchJob() *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})
	return o
}

// jobCompleteCondition checks a batch/v1 Job for the Complete condition.
func jobComplete(ctx context.Context, ns, name string) (bool, error) {
	job := &unstructured.Unstructured{}
	job.SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})
	if err := h.k8s.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, job); err != nil {
		return false, err
	}
	conds, found, _ := unstructured.NestedSlice(job.Object, "status", "conditions")
	if !found {
		return false, nil
	}
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] == "Complete" && m["status"] == "True" {
			return true, nil
		}
	}
	return false, nil
}
