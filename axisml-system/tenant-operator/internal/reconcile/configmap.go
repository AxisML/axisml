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

	axisml "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// ConfigMaps reconciles spec.initResources.configMaps[] (§6.5). Data is
// copied from sourceConfigMapRef on every reconcile (delayed by the
// resync period — see design §5). srcReader is uncached; the cached client
// only sees managed-by=tenant-operator objects, so source ConfigMaps must
// be read via APIReader.
func ConfigMaps(
	ctx context.Context,
	c client.Client,
	srcReader client.Reader,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
) ([]axisml.InitResourceItemStatus, error) {
	statuses := make([]axisml.InitResourceItemStatus, 0, len(t.Spec.InitResources.ConfigMaps))
	for _, cm := range t.Spec.InitResources.ConfigMaps {
		ready, msg, err := upsertCopiedConfigMap(ctx, c, srcReader, scheme, t, cm)
		statuses = append(statuses, axisml.InitResourceItemStatus{Name: cm.Name, Ready: ready, Message: msg})
		if err != nil {
			return statuses, err
		}
	}
	desired := nameSet(stringsFromConfigMaps(t.Spec.InitResources.ConfigMaps))
	if err := gcOrphanConfigMaps(ctx, c, t, desired); err != nil {
		return statuses, err
	}
	return statuses, nil
}

func upsertCopiedConfigMap(
	ctx context.Context,
	c client.Client,
	srcReader client.Reader,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	spec axisml.ConfigMapSpec,
) (bool, string, error) {
	src := spec.SourceConfigMapRef
	if src.Namespace == "" || src.Name == "" {
		return false, "sourceConfigMapRef.namespace and .name are required", nil
	}
	srcCM := &corev1.ConfigMap{}
	if err := srcReader.Get(ctx, types.NamespacedName{Namespace: src.Namespace, Name: src.Name}, srcCM); err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Sprintf("source configmap %s/%s not found", src.Namespace, src.Name), nil
		}
		return false, fmt.Sprintf("get source configmap failed: %v", err), err
	}

	name := PerTenantResourceName(t.Name, spec.Name)
	ns := t.Spec.Namespace.Name
	labels := ApplyTenantLabels(t, nil)

	existing := &corev1.ConfigMap{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, existing)
	switch {
	case apierrors.IsNotFound(getErr):
		out := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: labels},
			Data:       srcCM.Data,
			BinaryData: srcCM.BinaryData,
		}
		if err := controllerutil.SetControllerReference(t, out, scheme); err != nil {
			return false, fmt.Sprintf("set ownerRef failed: %v", err), err
		}
		if err := c.Create(ctx, out); err != nil {
			return false, fmt.Sprintf("create configmap failed: %v", err), err
		}
		return true, "", nil
	case getErr != nil:
		return false, fmt.Sprintf("get target configmap failed: %v", getErr), getErr
	default:
		base := existing.DeepCopy()
		needsPatch := false
		if !equality.Semantic.DeepEqual(existing.Data, srcCM.Data) {
			existing.Data = srcCM.Data
			needsPatch = true
		}
		if !equality.Semantic.DeepEqual(existing.BinaryData, srcCM.BinaryData) {
			existing.BinaryData = srcCM.BinaryData
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
				return false, fmt.Sprintf("patch configmap failed: %v", err), err
			}
		}
		return true, "", nil
	}
}

func gcOrphanConfigMaps(ctx context.Context, c client.Client, t *axisml.Tenant, desired map[string]struct{}) error {
	list := &corev1.ConfigMapList{}
	if err := c.List(ctx, list,
		client.InNamespace(t.Spec.Namespace.Name),
		client.MatchingLabels{axisml.LabelTenantID: t.Labels[axisml.LabelTenantID]},
	); err != nil {
		return fmt.Errorf("list configmaps: %w", err)
	}
	prefix := PerTenantResourceName(t.Name, "")
	for i := range list.Items {
		cm := &list.Items[i]
		if !hasOwner(cm.OwnerReferences, t) {
			continue
		}
		sub := stripPrefix(cm.Name, prefix)
		if _, keep := desired[sub]; keep {
			continue
		}
		if err := c.Delete(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete orphan configmap %s: %w", cm.Name, err)
		}
	}
	return nil
}

func stringsFromConfigMaps(in []axisml.ConfigMapSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}
