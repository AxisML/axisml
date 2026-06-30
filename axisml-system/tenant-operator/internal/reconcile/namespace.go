package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// Namespace ensures the target Namespace exists and carries the
// tenant.axisml.io/managed-by label. Behaviors per design §6.1:
//   - never delete (RBAC also blocks it)
//   - never overwrite existing labels/annotations on a shared namespace
//   - only stamp tenant.axisml.io/managed-by=tenant-operator if missing
//   - no ownerReference
func Namespace(ctx context.Context, c client.Client, t *axisml.Tenant) (ready bool, message string, err error) {
	name := t.Spec.Namespace.Name
	if name == "" {
		return false, "spec.namespace.name is empty", fmt.Errorf("namespace name empty")
	}

	ns := &corev1.Namespace{}
	getErr := c.Get(ctx, types.NamespacedName{Name: name}, ns)

	switch {
	case apierrors.IsNotFound(getErr):
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Labels:      withManagedByLabel(t.Spec.Namespace.Labels),
				Annotations: copyMap(t.Spec.Namespace.Annotations),
			},
		}
		if err := c.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
			return false, fmt.Sprintf("create namespace failed: %v", err), err
		}
	case getErr != nil:
		return false, fmt.Sprintf("get namespace failed: %v", getErr), getErr
	default:
		// Existing namespace: only stamp managed-by label if absent; never
		// touch other fields and never overwrite a foreign owner's value, so
		// we don't steal ownership of a shared namespace.
		if _, present := ns.Labels[axisml.LabelManagedBy]; !present {
			patch := client.MergeFrom(ns.DeepCopy())
			if ns.Labels == nil {
				ns.Labels = map[string]string{}
			}
			ns.Labels[axisml.LabelManagedBy] = axisml.ManagedByValue
			if err := c.Patch(ctx, ns, patch); err != nil {
				return false, fmt.Sprintf("label namespace failed: %v", err), err
			}
		}
	}

	// Refresh to read phase. A freshly created Namespace usually returns
	// phase=Active immediately because the apiserver synthesizes it.
	if err := c.Get(ctx, types.NamespacedName{Name: name}, ns); err != nil {
		return false, fmt.Sprintf("get namespace after upsert failed: %v", err), err
	}
	if ns.Status.Phase != corev1.NamespaceActive {
		return false, fmt.Sprintf("namespace phase=%s", ns.Status.Phase), nil
	}
	return true, "", nil
}

func withManagedByLabel(in map[string]string) map[string]string {
	out := copyMap(in)
	if out == nil {
		out = map[string]string{}
	}
	out[axisml.LabelManagedBy] = axisml.ManagedByValue
	return out
}

func copyMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
