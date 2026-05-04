// Package controller hosts the Tenant reconciler. It is intentionally thin:
// per design §5 the controller dispatches to subreconcilers, then derives a
// single status patch from their aggregated outcome.
package controller

import (
	"context"

	schedv1alpha1 "github.com/koordinator-sh/koordinator/apis/thirdparty/scheduler-plugins/pkg/apis/scheduling/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	axisml "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/tenant-operator/internal/reconcile"
	"github.com/axisml/axisml/components/operators/tenant-operator/internal/validate"
)

// TenantReconciler reconciles a Tenant CR.
type TenantReconciler struct {
	client.Client
	// APIReader is an uncached reader used for source Secret / ConfigMap
	// reads (design §5: source resources are not watched). Reading them
	// through the cached Client would either pull every Secret/ConfigMap in
	// the cluster into the cache, or — once the cache is label-restricted
	// to managed-by=tenant-operator — return NotFound because source
	// objects don't carry that label.
	APIReader    client.Reader
	Scheme       *runtime.Scheme
	ValidateOpts validate.Options
}

// SetupWithManager wires the watch topology described in design §5:
// Tenant is the primary; per-tenant resources are Owned (so ownerRef-driven
// events trigger reconcile, including ElasticQuota.status.used updates).
// Namespace is intentionally NOT owned — it has no ownerRef and is shared.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&axisml.Tenant{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&schedv1alpha1.ElasticQuota{}).
		Named("tenant").
		Complete(r)
}

// Reconcile implements the controller-runtime Reconciler interface.
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("tenant", req.Name)

	tenant := &axisml.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		if apierrors.IsNotFound(err) {
			// CR removed: nothing to do. K8s GC handles per-tenant resources
			// via ownerReference; Namespace is intentionally untouched.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	previousPhase := tenant.Status.Phase

	// Pure validation; failure → phase=Failed, no K8s mutations.
	if err := validate.ValidateMeta(&tenant.ObjectMeta); err != nil {
		logger.Info("tenant metadata invalid", "error", err.Error())
		return ctrl.Result{}, r.patchStatusFailed(ctx, tenant, err.Error())
	}
	if err := validate.Validate(&tenant.Spec, r.ValidateOpts); err != nil {
		logger.Info("tenant spec invalid", "error", err.Error())
		return ctrl.Result{}, r.patchStatusFailed(ctx, tenant, err.Error())
	}

	// Suspended is purely a phase signal — keep all underlying resources
	// running; Compute API enforces submission gating from
	// tenants.status='Suspended' (design §5 / §6.2.1). Carry forward the last
	// observed quota and init-resource statuses so suspension does not wipe
	// status.quotas[].used (which Compute caches) or per-item readiness.
	if tenant.Spec.Suspended {
		return ctrl.Result{}, r.patchStatus(ctx, tenant, reconcile.Aggregate{
			NamespaceReady:   tenant.Status.NamespaceReady,
			Quotas:           tenant.Status.Quotas,
			ImagePullSecrets: tenant.Status.InitResources.ImagePullSecrets,
			Secrets:          tenant.Status.InitResources.Secrets,
			ConfigMaps:       tenant.Status.InitResources.ConfigMaps,
			ServiceAccounts:  tenant.Status.InitResources.ServiceAccounts,
		}, axisml.TenantPhaseSuspended, "spec.suspended=true")
	}

	// Pre-populate the aggregate with the previously observed status so a
	// transient failure in one subreconciler doesn't wipe the per-item
	// readiness — and especially the cached ElasticQuota.status.used that
	// Compute consumes — for sections we never reach this pass. Each
	// subreconciler that does run overwrites its own field below.
	agg := reconcile.Aggregate{
		NamespaceReady:   tenant.Status.NamespaceReady,
		Quotas:           tenant.Status.Quotas,
		ImagePullSecrets: tenant.Status.InitResources.ImagePullSecrets,
		Secrets:          tenant.Status.InitResources.Secrets,
		ConfigMaps:       tenant.Status.InitResources.ConfigMaps,
		ServiceAccounts:  tenant.Status.InitResources.ServiceAccounts,
	}

	// 1. Namespace
	nsReady, nsMsg, nsErr := reconcile.Namespace(ctx, r.Client, tenant)
	agg.NamespaceReady = nsReady
	agg.NamespaceMsg = nsMsg
	if nsErr != nil {
		agg.CriticalFailure = true
		agg.FailureMessage = "namespace reconcile failed: " + nsErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, nsErr)
	}

	// 2. ElasticQuotas
	qStats, qErr := reconcile.ElasticQuotas(ctx, r.Client, r.Scheme, tenant)
	agg.Quotas = qStats
	if qErr != nil {
		agg.CriticalFailure = true
		agg.FailureMessage = "elasticquota reconcile failed: " + qErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, qErr)
	}

	// 3. ImagePullSecrets / Secrets / ConfigMaps / ServiceAccounts
	pulls, pullErr := reconcile.ImagePullSecrets(ctx, r.Client, r.APIReader, r.Scheme, tenant)
	agg.ImagePullSecrets = pulls
	if pullErr != nil {
		agg.FailureMessage = "imagePullSecrets reconcile failed: " + pullErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, pullErr)
	}
	secs, secErr := reconcile.Secrets(ctx, r.Client, r.APIReader, r.Scheme, tenant)
	agg.Secrets = secs
	if secErr != nil {
		agg.FailureMessage = "secrets reconcile failed: " + secErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, secErr)
	}
	cms, cmErr := reconcile.ConfigMaps(ctx, r.Client, r.APIReader, r.Scheme, tenant)
	agg.ConfigMaps = cms
	if cmErr != nil {
		agg.FailureMessage = "configMaps reconcile failed: " + cmErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, cmErr)
	}
	sas, saErr := reconcile.ServiceAccounts(ctx, r.Client, r.Scheme, tenant)
	agg.ServiceAccounts = sas
	if saErr != nil {
		agg.FailureMessage = "serviceAccounts reconcile failed: " + saErr.Error()
		return r.recordAndRequeue(ctx, tenant, agg, previousPhase, saErr)
	}

	phase, msg := reconcile.DerivePhase(tenant, agg, previousPhase)
	return ctrl.Result{}, r.patchStatus(ctx, tenant, agg, phase, msg)
}

func (r *TenantReconciler) recordAndRequeue(
	ctx context.Context,
	t *axisml.Tenant,
	agg reconcile.Aggregate,
	prev axisml.TenantPhase,
	cause error,
) (ctrl.Result, error) {
	phase, msg := reconcile.DerivePhase(t, agg, prev)
	if err := r.patchStatus(ctx, t, agg, phase, msg); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, cause
}

func (r *TenantReconciler) patchStatus(
	ctx context.Context,
	t *axisml.Tenant,
	agg reconcile.Aggregate,
	phase axisml.TenantPhase,
	message string,
) error {
	desired := reconcile.BuildStatus(t, agg, phase, message)
	patch := client.MergeFrom(t.DeepCopy())
	t.Status = desired
	return r.Status().Patch(ctx, t, patch)
}

// patchStatusFailed is the validation-failure path. It deliberately leaves
// Status.Quotas / InitResources / Conditions / NamespaceReady alone — the
// Reconcile never observed K8s state on this pass, so previously-known good
// per-item readiness shouldn't be wiped.
func (r *TenantReconciler) patchStatusFailed(
	ctx context.Context,
	t *axisml.Tenant,
	message string,
) error {
	patch := client.MergeFrom(t.DeepCopy())
	t.Status.ObservedGeneration = t.Generation
	t.Status.Phase = axisml.TenantPhaseFailed
	t.Status.Message = message
	return r.Status().Patch(ctx, t, patch)
}
