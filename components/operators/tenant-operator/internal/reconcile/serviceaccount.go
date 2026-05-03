package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	axisml "github.com/axisml-io/axisml/components/operators/tenant-operator/api/v1alpha1"
)

// ServiceAccounts reconciles spec.initResources.serviceAccounts[] together
// with their associated Role + RoleBinding when rbac is declared (§6.6).
//
// Two RBAC shapes:
//  1. rbac.rules + roleRef.kind != "ClusterRole"  → create Role + RoleBinding
//  2. rbac.roleRef.kind == "ClusterRole"          → only RoleBinding pointing
//     at the platform-provided ClusterRole
func ServiceAccounts(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
) ([]axisml.InitResourceItemStatus, error) {
	statuses := make([]axisml.InitResourceItemStatus, 0, len(t.Spec.InitResources.ServiceAccounts))
	for _, sa := range t.Spec.InitResources.ServiceAccounts {
		ready, msg, err := upsertSA(ctx, c, scheme, t, sa)
		statuses = append(statuses, axisml.InitResourceItemStatus{Name: sa.Name, Ready: ready, Message: msg})
		if err != nil {
			return statuses, err
		}
	}

	desired := nameSet(stringsFromSAs(t.Spec.InitResources.ServiceAccounts))
	if err := gcOrphanSAs(ctx, c, t, desired); err != nil {
		return statuses, err
	}
	return statuses, nil
}

func upsertSA(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	saSpec axisml.ServiceAccountSpec,
) (bool, string, error) {
	ns := t.Spec.Namespace.Name
	saName := PerTenantResourceName(t.Name, saSpec.Name)
	labels := ApplyTenantLabels(t, nil)

	// Resolve referenced ImagePullSecrets to their final per-tenant names.
	finalPullRefs := make([]corev1.LocalObjectReference, 0, len(saSpec.ImagePullSecrets))
	for _, ref := range saSpec.ImagePullSecrets {
		finalPullRefs = append(finalPullRefs, corev1.LocalObjectReference{
			Name: PerTenantResourceName(t.Name, ref),
		})
	}

	// 1. ServiceAccount
	if err := upsertServiceAccountObj(ctx, c, scheme, t, ns, saName, labels, finalPullRefs); err != nil {
		return false, fmt.Sprintf("upsert service account failed: %v", err), err
	}

	// 2. Optional Role + RoleBinding
	if saSpec.RBAC != nil {
		if msg, err := upsertRoleAndBinding(ctx, c, scheme, t, ns, saName, labels, saSpec.RBAC); err != nil {
			return false, msg, err
		}
	} else {
		// No RBAC declared: clean up any leftover Role/RoleBinding from a
		// previous spec that did declare it.
		if err := deleteIfExists(ctx, c, &rbacv1.RoleBinding{}, ns, saName); err != nil {
			return false, fmt.Sprintf("cleanup rolebinding failed: %v", err), err
		}
		if err := deleteIfExists(ctx, c, &rbacv1.Role{}, ns, saName); err != nil {
			return false, fmt.Sprintf("cleanup role failed: %v", err), err
		}
	}

	return true, "", nil
}

func upsertServiceAccountObj(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	ns, name string,
	labels map[string]string,
	pullRefs []corev1.LocalObjectReference,
) error {
	existing := &corev1.ServiceAccount{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, existing)
	switch {
	case apierrors.IsNotFound(getErr):
		out := &corev1.ServiceAccount{
			ObjectMeta:       metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
			ImagePullSecrets: pullRefs,
		}
		if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
			return err
		}
		return c.Create(ctx, out)
	case getErr != nil:
		return getErr
	default:
		base := existing.DeepCopy()
		needsPatch := false
		if !equality.Semantic.DeepEqual(existing.ImagePullSecrets, pullRefs) {
			existing.ImagePullSecrets = pullRefs
			needsPatch = true
		}
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		for k, v := range labels {
			if existing.Labels[k] != v {
				existing.Labels[k] = v
				needsPatch = true
			}
		}
		if !hasOwner(existing.OwnerReferences, t) {
			if err := controllerutil.SetControllerReference(t, existing, scheme); err != nil {
				return err
			}
			needsPatch = true
		}
		if needsPatch {
			return c.Patch(ctx, existing, client.MergeFrom(base))
		}
		return nil
	}
}

func upsertRoleAndBinding(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	ns, saName string,
	labels map[string]string,
	rbac *axisml.RBACSpec,
) (string, error) {
	roleName := saName
	useClusterRole := rbac.RoleRef != nil && rbac.RoleRef.Kind == "ClusterRole"

	if !useClusterRole {
		// Build/upsert the Role from rbac.rules.
		if err := upsertRole(ctx, c, scheme, t, ns, roleName, labels, rbac.Rules); err != nil {
			return fmt.Sprintf("upsert role failed: %v", err), err
		}
	} else {
		// Caller chose to bind to an existing ClusterRole; remove any leftover
		// per-tenant Role from a previous spec.
		if err := deleteIfExists(ctx, c, &rbacv1.Role{}, ns, roleName); err != nil {
			return fmt.Sprintf("cleanup stale role failed: %v", err), err
		}
	}

	// RoleBinding always exists when rbac is declared.
	if err := upsertRoleBinding(ctx, c, scheme, t, ns, saName, labels, rbac, useClusterRole); err != nil {
		return fmt.Sprintf("upsert rolebinding failed: %v", err), err
	}
	return "", nil
}

func upsertRole(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	ns, name string,
	labels map[string]string,
	rules []rbacv1.PolicyRule,
) error {
	existing := &rbacv1.Role{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, existing)
	switch {
	case apierrors.IsNotFound(getErr):
		out := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
			Rules:      rules,
		}
		if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
			return err
		}
		return c.Create(ctx, out)
	case getErr != nil:
		return getErr
	default:
		base := existing.DeepCopy()
		needsPatch := false
		if !equality.Semantic.DeepEqual(existing.Rules, rules) {
			existing.Rules = rules
			needsPatch = true
		}
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		for k, v := range labels {
			if existing.Labels[k] != v {
				existing.Labels[k] = v
				needsPatch = true
			}
		}
		if !hasOwner(existing.OwnerReferences, t) {
			if err := controllerutil.SetControllerReference(t, existing, scheme); err != nil {
				return err
			}
			needsPatch = true
		}
		if needsPatch {
			return c.Patch(ctx, existing, client.MergeFrom(base))
		}
		return nil
	}
}

func upsertRoleBinding(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	ns, saName string,
	labels map[string]string,
	rbac *axisml.RBACSpec,
	useClusterRole bool,
) error {
	desiredRoleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     saName, // default: per-tenant Role created above
	}
	if useClusterRole {
		desiredRoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     rbac.RoleRef.Name,
		}
	}
	desiredSubjects := []rbacv1.Subject{{
		Kind:      "ServiceAccount",
		Name:      saName,
		Namespace: ns,
	}}

	existing := &rbacv1.RoleBinding{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: saName}, existing)
	switch {
	case apierrors.IsNotFound(getErr):
		out := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns, Labels: labels},
			Subjects:   desiredSubjects,
			RoleRef:    desiredRoleRef,
		}
		if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
			return err
		}
		return c.Create(ctx, out)
	case getErr != nil:
		return getErr
	default:
		// RoleRef is immutable on RoleBinding — if it changed we must
		// delete-and-recreate.
		if !equality.Semantic.DeepEqual(existing.RoleRef, desiredRoleRef) {
			if err := c.Delete(ctx, existing); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			out := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns, Labels: labels},
				Subjects:   desiredSubjects,
				RoleRef:    desiredRoleRef,
			}
			if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
				return err
			}
			return c.Create(ctx, out)
		}
		base := existing.DeepCopy()
		needsPatch := false
		if !equality.Semantic.DeepEqual(existing.Subjects, desiredSubjects) {
			existing.Subjects = desiredSubjects
			needsPatch = true
		}
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		for k, v := range labels {
			if existing.Labels[k] != v {
				existing.Labels[k] = v
				needsPatch = true
			}
		}
		if !hasOwner(existing.OwnerReferences, t) {
			if err := controllerutil.SetControllerReference(t, existing, scheme); err != nil {
				return err
			}
			needsPatch = true
		}
		if needsPatch {
			return c.Patch(ctx, existing, client.MergeFrom(base))
		}
		return nil
	}
}

func deleteIfExists(ctx context.Context, c client.Client, obj client.Object, ns, name string) error {
	obj.SetNamespace(ns)
	obj.SetName(name)
	if err := c.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func gcOrphanSAs(ctx context.Context, c client.Client, t *axisml.Tenant, desired map[string]struct{}) error {
	list := &corev1.ServiceAccountList{}
	if err := c.List(ctx, list,
		client.InNamespace(t.Spec.Namespace.Name),
		client.MatchingLabels{axisml.LabelTenantID: t.Labels[axisml.LabelTenantID]},
	); err != nil {
		return fmt.Errorf("list service accounts: %w", err)
	}
	prefix := PerTenantResourceName(t.Name, "")
	for i := range list.Items {
		sa := &list.Items[i]
		if !hasOwner(sa.OwnerReferences, t) {
			continue
		}
		sub := stripPrefix(sa.Name, prefix)
		if _, keep := desired[sub]; keep {
			continue
		}
		// Delete RoleBinding + Role first (best-effort; ownerReference would
		// also handle it on Tenant deletion, but here the Tenant survives).
		_ = deleteIfExists(ctx, c, &rbacv1.RoleBinding{}, sa.Namespace, sa.Name)
		_ = deleteIfExists(ctx, c, &rbacv1.Role{}, sa.Namespace, sa.Name)
		if err := c.Delete(ctx, sa); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete orphan service account %s: %w", sa.Name, err)
		}
	}
	return nil
}

func stringsFromSAs(in []axisml.ServiceAccountSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}
