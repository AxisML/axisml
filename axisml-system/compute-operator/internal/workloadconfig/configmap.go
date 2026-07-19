// Package workloadconfig reconciles configuration resources shared by the
// MLRun and MLService dispatchers.
package workloadconfig

import (
	"context"
	"errors"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configapi "github.com/axisml/axisml/axisml-system/apis/pkg/workloadconfig"
)

const (
	LabelManagedConfig = "compute.axisml.io/managed-configmap"
	LabelOwnerUID      = "compute.axisml.io/config-owner-uid"
	ManagedValue       = "true"
)

// OwnershipConflictError reports a deterministic name collision with an
// existing ConfigMap that this workload does not control.
type OwnershipConflictError struct {
	Namespace string
	Name      string
	OwnerKind string
	OwnerName string
}

func (e *OwnershipConflictError) Error() string {
	return fmt.Sprintf("ConfigMap %s/%s already exists and is not owned by %s %s",
		e.Namespace, e.Name, e.OwnerKind, e.OwnerName)
}

// IsOwnershipConflict distinguishes a terminal contract conflict from
// transient Kubernetes API failures that controller-runtime should retry.
func IsOwnershipConflict(err error) bool {
	var conflict *OwnershipConflictError
	return errors.As(err, &conflict)
}

// Validate checks the pure, cluster-independent ConfigMap contract.
func Validate(configMaps []configapi.ConfigMap, path *field.Path) field.ErrorList {
	var errs field.ErrorList
	seen := make(map[string]struct{}, len(configMaps))
	for i, configMap := range configMaps {
		itemPath := path.Index(i)
		if configMap.Name == "" {
			errs = append(errs, field.Required(itemPath.Child("name"), "name is required"))
		} else {
			for _, msg := range validation.IsDNS1123Subdomain(configMap.Name) {
				errs = append(errs, field.Invalid(itemPath.Child("name"), configMap.Name, msg))
			}
			if _, duplicate := seen[configMap.Name]; duplicate {
				errs = append(errs, field.Duplicate(itemPath.Child("name"), configMap.Name))
			}
			seen[configMap.Name] = struct{}{}
		}
		for key := range configMap.Data {
			for _, msg := range validation.IsConfigMapKey(key) {
				errs = append(errs, field.Invalid(itemPath.Child("data").Key(key), key, msg))
			}
		}
	}
	return errs
}

// Reconcile creates or drift-corrects the workload-owned ConfigMaps and
// removes obsolete ones left by an initial spec that changed before the
// workload's immutable-spec baseline was recorded.
func Reconcile(
	ctx context.Context,
	c client.Client,
	owner client.Object,
	ownerGVK schema.GroupVersionKind,
	configMaps []configapi.ConfigMap,
	extraLabels map[string]string,
) error {
	desiredNames := make(map[string]struct{}, len(configMaps))
	for _, spec := range configMaps {
		desiredNames[spec.Name] = struct{}{}
		if err := upsert(ctx, c, owner, ownerGVK, spec, extraLabels); err != nil {
			return err
		}
	}
	return deleteObsolete(ctx, c, owner, desiredNames)
}

func upsert(
	ctx context.Context,
	c client.Client,
	owner client.Object,
	ownerGVK schema.GroupVersionKind,
	spec configapi.ConfigMap,
	extraLabels map[string]string,
) error {
	key := client.ObjectKey{Namespace: owner.GetNamespace(), Name: spec.Name}
	current := &corev1.ConfigMap{}
	err := c.Get(ctx, key, current)
	if apierrors.IsNotFound(err) {
		desired := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            spec.Name,
				Namespace:       owner.GetNamespace(),
				Labels:          managedLabels(owner, extraLabels),
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(owner, ownerGVK)},
			},
			Data: maps.Clone(spec.Data),
		}
		if err := c.Create(ctx, desired); err != nil {
			return fmt.Errorf("create workload ConfigMap %s/%s: %w", key.Namespace, key.Name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get workload ConfigMap %s/%s: %w", key.Namespace, key.Name, err)
	}

	controller := metav1.GetControllerOf(current)
	if controller == nil || controller.UID != owner.GetUID() {
		return &OwnershipConflictError{
			Namespace: key.Namespace,
			Name:      key.Name,
			OwnerKind: ownerGVK.Kind,
			OwnerName: owner.GetName(),
		}
	}

	before := current.DeepCopy()
	current.Data = maps.Clone(spec.Data)
	if current.Labels == nil {
		current.Labels = map[string]string{}
	}
	for key, value := range managedLabels(owner, extraLabels) {
		current.Labels[key] = value
	}
	if equality.Semantic.DeepEqual(before.Data, current.Data) &&
		equality.Semantic.DeepEqual(before.Labels, current.Labels) {
		return nil
	}
	if err := c.Patch(ctx, current, client.MergeFrom(before)); err != nil {
		return fmt.Errorf("patch workload ConfigMap %s/%s: %w", key.Namespace, key.Name, err)
	}
	return nil
}

func deleteObsolete(ctx context.Context, c client.Client, owner client.Object, desiredNames map[string]struct{}) error {
	list := &corev1.ConfigMapList{}
	if err := c.List(ctx, list,
		client.InNamespace(owner.GetNamespace()),
		client.MatchingLabels{
			LabelManagedConfig: ManagedValue,
			LabelOwnerUID:      string(owner.GetUID()),
		},
	); err != nil {
		return fmt.Errorf("list workload ConfigMaps: %w", err)
	}
	for i := range list.Items {
		configMap := &list.Items[i]
		if _, keep := desiredNames[configMap.Name]; keep || !metav1.IsControlledBy(configMap, owner) {
			continue
		}
		if err := c.Delete(ctx, configMap); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete obsolete workload ConfigMap %s/%s: %w",
				configMap.Namespace, configMap.Name, err)
		}
	}
	return nil
}

func managedLabels(owner client.Object, extra map[string]string) map[string]string {
	labels := map[string]string{
		LabelManagedConfig: ManagedValue,
		LabelOwnerUID:      string(owner.GetUID()),
	}
	for key, value := range extra {
		if value != "" {
			labels[key] = value
		}
	}
	return labels
}
