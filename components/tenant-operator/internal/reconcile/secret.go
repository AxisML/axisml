package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	axisml "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// ImagePullSecrets reconciles spec.initResources.imagePullSecrets[] (§6.3).
// Each entry is rendered as a Secret of type kubernetes.io/dockerconfigjson
// in the target namespace, copied from the source secret on every reconcile.
// srcReader is an uncached reader; the cached client is restricted to
// managed-by=tenant-operator objects so source secrets aren't visible there.
func ImagePullSecrets(
	ctx context.Context,
	c client.Client,
	srcReader client.Reader,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
) ([]axisml.InitResourceItemStatus, error) {
	statuses := make([]axisml.InitResourceItemStatus, 0, len(t.Spec.InitResources.ImagePullSecrets))
	for _, s := range t.Spec.InitResources.ImagePullSecrets {
		ready, msg, err := upsertCopiedSecret(ctx, c, srcReader, scheme, t, s.Name, corev1.SecretTypeDockerConfigJson, secretRoleImagePull, s.SourceSecretRef)
		statuses = append(statuses, axisml.InitResourceItemStatus{Name: s.Name, Ready: ready, Message: msg})
		if err != nil {
			return statuses, err
		}
	}

	desired := nameSet(stringsFromPullSecrets(t.Spec.InitResources.ImagePullSecrets))
	if err := gcOrphanSecrets(ctx, c, t, desired, secretRoleImagePull); err != nil {
		return statuses, err
	}
	return statuses, nil
}

// Secrets reconciles spec.initResources.secrets[] (§6.4). Type defaults to
// Opaque; user-set type is honored even when it disagrees with the source
// (the message is recorded but the spec wins). Type changes force a delete +
// recreate because Secret.type is immutable.
func Secrets(
	ctx context.Context,
	c client.Client,
	srcReader client.Reader,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
) ([]axisml.InitResourceItemStatus, error) {
	statuses := make([]axisml.InitResourceItemStatus, 0, len(t.Spec.InitResources.Secrets))
	for _, s := range t.Spec.InitResources.Secrets {
		stype := s.Type
		if stype == "" {
			stype = corev1.SecretTypeOpaque
		}
		ready, msg, err := upsertCopiedSecret(ctx, c, srcReader, scheme, t, s.Name, stype, secretRoleGeneric, s.SourceSecretRef)
		statuses = append(statuses, axisml.InitResourceItemStatus{Name: s.Name, Ready: ready, Message: msg})
		if err != nil {
			return statuses, err
		}
	}

	desired := nameSet(stringsFromSecrets(t.Spec.InitResources.Secrets))
	if err := gcOrphanSecrets(ctx, c, t, desired, secretRoleGeneric); err != nil {
		return statuses, err
	}
	return statuses, nil
}

const (
	// labelSecretRole distinguishes ImagePullSecret vs generic Secret on
	// owner-side GC. Without it we'd mix the two kinds in nameSet diffs.
	labelSecretRole     = "axisml.io/secret-role"
	secretRoleImagePull = "imagepull"
	secretRoleGeneric   = "generic"
)

func upsertCopiedSecret(
	ctx context.Context,
	c client.Client,
	srcReader client.Reader,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	subName string,
	desiredType corev1.SecretType,
	role string,
	src axisml.SourceSecretRef,
) (bool, string, error) {
	if src.Namespace == "" || src.Name == "" {
		return false, "sourceSecretRef.namespace and .name are required", nil
	}
	srcSecret := &corev1.Secret{}
	if err := srcReader.Get(ctx, types.NamespacedName{Namespace: src.Namespace, Name: src.Name}, srcSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Sprintf("source secret %s/%s not found", src.Namespace, src.Name), nil
		}
		return false, fmt.Sprintf("get source secret failed: %v", err), err
	}

	name := PerTenantResourceName(t.Name, subName)
	ns := t.Spec.Namespace.Name
	labels := ApplyTenantLabels(t, map[string]string{labelSecretRole: role})

	existing := &corev1.Secret{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, existing)

	switch {
	case apierrors.IsNotFound(getErr):
		out := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
			Type:       desiredType,
			Data:       srcSecret.Data,
		}
		if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
			return false, fmt.Sprintf("set ownerRef failed: %v", err), err
		}
		if err := c.Create(ctx, out); err != nil {
			return false, fmt.Sprintf("create secret failed: %v", err), err
		}
		return true, warnIfTypeMismatch(srcSecret.Type, desiredType), nil
	case getErr != nil:
		return false, fmt.Sprintf("get target secret failed: %v", getErr), getErr
	default:
		// Type changed → must delete+recreate (Secret type is immutable).
		// Normalize empty Type to Opaque first: a Secret created with no
		// explicit type can come back from the apiserver as either "" or
		// "Opaque" depending on serializer path; treating "" as Opaque
		// avoids a recreate loop when desiredType is Opaque.
		existingType := existing.Type
		if existingType == "" {
			existingType = corev1.SecretTypeOpaque
		}
		if existingType != desiredType {
			if err := c.Delete(ctx, existing); err != nil && !apierrors.IsNotFound(err) {
				return false, fmt.Sprintf("delete on type change failed: %v", err), err
			}
			out := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
				Type:       desiredType,
				Data:       srcSecret.Data,
			}
			if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
				return false, fmt.Sprintf("set ownerRef failed: %v", err), err
			}
			if err := c.Create(ctx, out); err != nil {
				return false, fmt.Sprintf("recreate after type change failed: %v", err), err
			}
			return true, warnIfTypeMismatch(srcSecret.Type, desiredType), nil
		}

		base := existing.DeepCopy()
		needsPatch := false
		if !equality.Semantic.DeepEqual(existing.Data, srcSecret.Data) {
			existing.Data = srcSecret.Data
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
				return false, fmt.Sprintf("set ownerRef failed: %v", err), err
			}
			needsPatch = true
		}
		if needsPatch {
			if err := c.Patch(ctx, existing, client.MergeFrom(base)); err != nil {
				return false, fmt.Sprintf("patch secret failed: %v", err), err
			}
		}
		return true, warnIfTypeMismatch(srcSecret.Type, desiredType), nil
	}
}

func warnIfTypeMismatch(srcType, desired corev1.SecretType) string {
	if srcType != "" && srcType != desired {
		return fmt.Sprintf("source secret type=%s differs from spec.type=%s; using spec.type",
			srcType, desired)
	}
	return ""
}

func gcOrphanSecrets(ctx context.Context, c client.Client, t *axisml.Tenant, desiredSubNames map[string]struct{}, role string) error {
	list := &corev1.SecretList{}
	if err := c.List(ctx, list,
		client.InNamespace(t.Spec.Namespace.Name),
		client.MatchingLabels{
			axisml.LabelTenantID: t.Labels[axisml.LabelTenantID],
			labelSecretRole:      role,
		},
	); err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}
	for i := range list.Items {
		s := &list.Items[i]
		if !hasOwner(s.OwnerReferences, t) {
			continue
		}
		sub := stripPrefix(s.Name, PerTenantResourceName(t.Name, ""))
		if _, keep := desiredSubNames[sub]; keep {
			continue
		}
		if err := c.Delete(ctx, s); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete orphan secret %s: %w", s.Name, err)
		}
	}
	return nil
}

func stripPrefix(s, prefix string) string {
	if len(prefix) > 0 && len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func nameSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, n := range in {
		out[n] = struct{}{}
	}
	return out
}

func stringsFromPullSecrets(in []axisml.ImagePullSecretSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}

func stringsFromSecrets(in []axisml.SecretSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}
