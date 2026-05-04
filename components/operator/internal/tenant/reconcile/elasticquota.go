package reconcile

import (
	"context"
	"fmt"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	axisml "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"
)

// ElasticQuotas reconciles spec.quotas[] 1:1 to upstream
// scheduling.sigs.k8s.io/v1alpha1 ElasticQuota CRs in the tenant namespace.
//
// Per design §6.2:
//   - missing entries get Created, removed entries get Deleted, drift on
//     min/max gets Patched
//   - status.used is read-only: we copy ElasticQuota.status.used back into
//     QuotaStatus.Used; we never write spec.used or anything similar
func ElasticQuotas(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
) ([]axisml.QuotaStatus, error) {
	targetNS := t.Spec.Namespace.Name

	desired := make(map[string]axisml.QuotaSpec, len(t.Spec.Quotas))
	for _, q := range t.Spec.Quotas {
		desired[ElasticQuotaName(t.Name, q.Pool, q.Name)] = q
	}

	if err := gcOrphanElasticQuotas(ctx, c, t, targetNS, desired); err != nil {
		return nil, err
	}

	statuses := make([]axisml.QuotaStatus, 0, len(t.Spec.Quotas))
	for _, q := range t.Spec.Quotas {
		eqName := ElasticQuotaName(t.Name, q.Pool, q.Name)
		ready, used, msg, err := upsertElasticQuota(ctx, c, scheme, t, targetNS, eqName, q)
		statuses = append(statuses, axisml.QuotaStatus{
			Pool:    q.Pool,
			Name:    q.Name,
			Ready:   ready,
			Used:    used,
			Message: msg,
		})
		if err != nil {
			return statuses, err
		}
	}
	return statuses, nil
}

func upsertElasticQuota(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	t *axisml.Tenant,
	ns, name string,
	q axisml.QuotaSpec,
) (ready bool, used corev1.ResourceList, message string, err error) {
	eq := &schedv1alpha1.ElasticQuota{}
	getErr := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, eq)
	switch {
	case apierrors.IsNotFound(getErr):
		eq = &schedv1alpha1.ElasticQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    TenantLabels(t),
			},
			Spec: schedv1alpha1.ElasticQuotaSpec{
				Min: q.Min,
				Max: q.Max,
			},
		}
		if err := controllerutil.SetControllerReference(t, eq, scheme); err != nil {
			return false, nil, fmt.Sprintf("set ownerRef failed: %v", err), err
		}
		if err := c.Create(ctx, eq); err != nil {
			return false, nil, fmt.Sprintf("create elasticquota failed: %v", err), err
		}
	case getErr != nil:
		return false, nil, fmt.Sprintf("get elasticquota failed: %v", getErr), getErr
	default:
		needsPatch := false
		base := eq.DeepCopy()
		if !equality.Semantic.DeepEqual(eq.Spec.Min, q.Min) ||
			!equality.Semantic.DeepEqual(eq.Spec.Max, q.Max) {
			eq.Spec.Min = q.Min
			eq.Spec.Max = q.Max
			needsPatch = true
		}
		if eq.Labels == nil {
			eq.Labels = map[string]string{}
		}
		for k, v := range TenantLabels(t) {
			if eq.Labels[k] != v {
				eq.Labels[k] = v
				needsPatch = true
			}
		}
		if !hasOwner(eq.OwnerReferences, t) {
			if err := controllerutil.SetControllerReference(t, eq, scheme); err != nil {
				return false, nil, fmt.Sprintf("set ownerRef failed: %v", err), err
			}
			needsPatch = true
		}
		if needsPatch {
			if err := c.Patch(ctx, eq, client.MergeFrom(base)); err != nil {
				return false, nil, fmt.Sprintf("patch elasticquota failed: %v", err), err
			}
		}
	}

	if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, eq); err != nil {
		return false, nil, fmt.Sprintf("get elasticquota after upsert failed: %v", err), err
	}
	return true, eq.Status.Used.DeepCopy(), "", nil
}

func hasOwner(refs []metav1.OwnerReference, t *axisml.Tenant) bool {
	for _, r := range refs {
		if r.UID == t.UID {
			return true
		}
	}
	return false
}

// gcOrphanElasticQuotas deletes ElasticQuotas owned by this Tenant in the
// target namespace whose object name is not in `desired`.
func gcOrphanElasticQuotas(
	ctx context.Context,
	c client.Client,
	t *axisml.Tenant,
	ns string,
	desired map[string]axisml.QuotaSpec,
) error {
	list := &schedv1alpha1.ElasticQuotaList{}
	if err := c.List(ctx, list,
		client.InNamespace(ns),
		client.MatchingLabels{axisml.LabelTenantID: t.Labels[axisml.LabelTenantID]},
	); err != nil {
		return fmt.Errorf("list elasticquotas: %w", err)
	}
	for i := range list.Items {
		eq := &list.Items[i]
		if _, keep := desired[eq.Name]; keep {
			continue
		}
		if !hasOwner(eq.OwnerReferences, t) {
			continue
		}
		if err := c.Delete(ctx, eq); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete orphan elasticquota %s: %w", eq.Name, err)
		}
	}
	return nil
}
