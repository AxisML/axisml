// Package kuberuntime implements the published ComputeRuntime contract over a
// real Kubernetes apiserver. It wraps a controller-runtime client.Client (typed
// CR CRUD + cached lists) and a client-go clientset (untyped sub-resources such
// as Pod log streaming); compute-operator remains the downstream consumer that
// maps the CRs onto Job / Deployment / StatefulSet / Service / HTTPRoute.
//
// This adapter is the reference implementation the Standalone runtime
// in the axisml-lite repo mirrors: both speak the same MLRun / MLService /
// MLTrafficPolicy API types, CR Status semantics and AxisML labels.
package kuberuntime

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	mltrafficpolicyv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/components/compute-service/pkg/extensions"
)

// KubernetesRuntime implements extensions.ComputeRuntime against an
// apiserver.
type KubernetesRuntime struct {
	ctrl      client.Client
	clientset kubernetes.Interface
}

var _ extensions.ComputeRuntime = (*KubernetesRuntime)(nil)

// New builds a KubernetesRuntime. The ctrl client serves typed CR CRUD and
// cached Pod / Event lists; clientset serves Pod log streaming.
func New(ctrl client.Client, clientset kubernetes.Interface) *KubernetesRuntime {
	return &KubernetesRuntime{ctrl: ctrl, clientset: clientset}
}

// ----------------------------------------------------------------------
// MLRun

// ApplyMLRun creates the MLRun when absent. A Run's spec is immutable once
// created, so applying an already-present Run is an idempotent no-op.
func (r *KubernetesRuntime) ApplyMLRun(ctx context.Context, desired *mlrunv1alpha1.MLRun) error {
	cur := &mlrunv1alpha1.MLRun{}
	err := r.ctrl.Get(ctx, client.ObjectKeyFromObject(desired), cur)
	if apierrors.IsNotFound(err) {
		return r.ctrl.Create(ctx, desired)
	}
	return err
}

// ObserveMLRun returns the current MLRun CR Status. A missing CR surfaces as a
// NotFound error recognizable by apierrors.IsNotFound.
func (r *KubernetesRuntime) ObserveMLRun(ctx context.Context, key types.NamespacedName) (mlrunv1alpha1.MLRunStatus, error) {
	cr := &mlrunv1alpha1.MLRun{}
	if err := r.ctrl.Get(ctx, key, cr); err != nil {
		return mlrunv1alpha1.MLRunStatus{}, err
	}
	return cr.Status, nil
}

// CancelMLRun signals cancellation by patching spec.runPolicy.suspend=true.
// Idempotent: a missing CR is treated as already cancelled.
func (r *KubernetesRuntime) CancelMLRun(ctx context.Context, key types.NamespacedName) error {
	cr := &mlrunv1alpha1.MLRun{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
	patch := client.RawPatch(types.MergePatchType, []byte(`{"spec":{"runPolicy":{"suspend":true}}}`))
	return ignoreNotFound(r.ctrl.Patch(ctx, cr, patch))
}

// DeleteMLRun removes the MLRun. Idempotent on an already-deleted CR.
func (r *KubernetesRuntime) DeleteMLRun(ctx context.Context, key types.NamespacedName) error {
	cr := &mlrunv1alpha1.MLRun{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
	return ignoreNotFound(r.ctrl.Delete(ctx, cr))
}

// ListMLRunInstances lists the Pods that carry the Run's stable id label. A Run
// whose CR has not yet been materialized (no instances exist yet) yields an
// empty list rather than a NotFound.
func (r *KubernetesRuntime) ListMLRunInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error) {
	id, err := r.runLabelID(ctx, key)
	if apierrors.IsNotFound(err) {
		return &corev1.PodList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return r.listPodsByLabel(ctx, key.Namespace, mlrunv1alpha1.LabelRunID, id)
}

// GetMLRunInstanceLogs streams the named instance's log after verifying it
// belongs to the addressed Run.
func (r *KubernetesRuntime) GetMLRunInstanceLogs(ctx context.Context, key types.NamespacedName, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	id, err := r.runLabelID(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := r.verifyInstance(ctx, key.Namespace, instance, mlrunv1alpha1.LabelRunID, id); err != nil {
		return nil, err
	}
	return r.streamLogs(ctx, key.Namespace, instance, opts)
}

// GetMLRunInstanceEvents returns events regarding the named instance (Pod)
// after verifying it belongs to the addressed Run.
func (r *KubernetesRuntime) GetMLRunInstanceEvents(ctx context.Context, key types.NamespacedName, instance string) (*eventsv1.EventList, error) {
	id, err := r.runLabelID(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := r.verifyInstance(ctx, key.Namespace, instance, mlrunv1alpha1.LabelRunID, id); err != nil {
		return nil, err
	}
	return r.eventsFor(ctx, key.Namespace, eventTarget{"Pod", instance})
}

// GetMLRunEvents returns events regarding the MLRun CR and its peer scheduling
// primitive (PodGroup).
func (r *KubernetesRuntime) GetMLRunEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.eventsFor(ctx, key.Namespace,
		eventTarget{"MLRun", key.Name},
		eventTarget{"PodGroup", key.Name},
	)
}

// ----------------------------------------------------------------------
// MLService

// ApplyMLService creates the MLService when absent, else converges its spec
// onto the desired one. Idempotent.
func (r *KubernetesRuntime) ApplyMLService(ctx context.Context, desired *mlservicev1alpha1.MLService) error {
	cur := &mlservicev1alpha1.MLService{}
	err := r.ctrl.Get(ctx, client.ObjectKeyFromObject(desired), cur)
	if apierrors.IsNotFound(err) {
		return r.ctrl.Create(ctx, desired)
	}
	if err != nil {
		return err
	}
	cur.Spec = desired.Spec
	mergeLabels(cur, desired.Labels)
	return r.ctrl.Update(ctx, cur)
}

func (r *KubernetesRuntime) ObserveMLService(ctx context.Context, key types.NamespacedName) (mlservicev1alpha1.MLServiceStatus, error) {
	cr := &mlservicev1alpha1.MLService{}
	if err := r.ctrl.Get(ctx, key, cr); err != nil {
		return mlservicev1alpha1.MLServiceStatus{}, err
	}
	return cr.Status, nil
}

func (r *KubernetesRuntime) DeleteMLService(ctx context.Context, key types.NamespacedName) error {
	cr := &mlservicev1alpha1.MLService{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
	return ignoreNotFound(r.ctrl.Delete(ctx, cr))
}

// ListMLServiceInstances lists the Pods that carry the Service's stable id
// label. A Service whose CR has not yet been materialized yields an empty list
// rather than a NotFound.
func (r *KubernetesRuntime) ListMLServiceInstances(ctx context.Context, key types.NamespacedName) (*corev1.PodList, error) {
	id, err := r.serviceLabelID(ctx, key)
	if apierrors.IsNotFound(err) {
		return &corev1.PodList{}, nil
	}
	if err != nil {
		return nil, err
	}
	return r.listPodsByLabel(ctx, key.Namespace, mlservicev1alpha1.LabelServiceID, id)
}

func (r *KubernetesRuntime) GetMLServiceInstanceLogs(ctx context.Context, key types.NamespacedName, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	id, err := r.serviceLabelID(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := r.verifyInstance(ctx, key.Namespace, instance, mlservicev1alpha1.LabelServiceID, id); err != nil {
		return nil, err
	}
	return r.streamLogs(ctx, key.Namespace, instance, opts)
}

func (r *KubernetesRuntime) GetMLServiceInstanceEvents(ctx context.Context, key types.NamespacedName, instance string) (*eventsv1.EventList, error) {
	id, err := r.serviceLabelID(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := r.verifyInstance(ctx, key.Namespace, instance, mlservicev1alpha1.LabelServiceID, id); err != nil {
		return nil, err
	}
	return r.eventsFor(ctx, key.Namespace, eventTarget{"Pod", instance})
}

// GetMLServiceEvents returns events regarding the MLService CR and the workload
// primitives compute-operator derives from it.
func (r *KubernetesRuntime) GetMLServiceEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.eventsFor(ctx, key.Namespace,
		eventTarget{"MLService", key.Name},
		eventTarget{"Deployment", key.Name},
		eventTarget{"StatefulSet", key.Name},
		eventTarget{"HTTPRoute", key.Name},
	)
}

// ----------------------------------------------------------------------
// MLTrafficPolicy

func (r *KubernetesRuntime) ApplyMLTrafficPolicy(ctx context.Context, desired *mltrafficpolicyv1alpha1.MLTrafficPolicy) error {
	cur := &mltrafficpolicyv1alpha1.MLTrafficPolicy{}
	err := r.ctrl.Get(ctx, client.ObjectKeyFromObject(desired), cur)
	if apierrors.IsNotFound(err) {
		return r.ctrl.Create(ctx, desired)
	}
	if err != nil {
		return err
	}
	cur.Spec = desired.Spec
	mergeLabels(cur, desired.Labels)
	return r.ctrl.Update(ctx, cur)
}

func (r *KubernetesRuntime) ObserveMLTrafficPolicy(ctx context.Context, key types.NamespacedName) (mltrafficpolicyv1alpha1.MLTrafficPolicyStatus, error) {
	cr := &mltrafficpolicyv1alpha1.MLTrafficPolicy{}
	if err := r.ctrl.Get(ctx, key, cr); err != nil {
		return mltrafficpolicyv1alpha1.MLTrafficPolicyStatus{}, err
	}
	return cr.Status, nil
}

func (r *KubernetesRuntime) DeleteMLTrafficPolicy(ctx context.Context, key types.NamespacedName) error {
	cr := &mltrafficpolicyv1alpha1.MLTrafficPolicy{ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace}}
	return ignoreNotFound(r.ctrl.Delete(ctx, cr))
}

// GetMLTrafficPolicyEvents returns events regarding the policy CR and its
// derived gateway route. MLTrafficPolicy has no instances.
func (r *KubernetesRuntime) GetMLTrafficPolicyEvents(ctx context.Context, key types.NamespacedName) (*eventsv1.EventList, error) {
	return r.eventsFor(ctx, key.Namespace,
		eventTarget{"MLTrafficPolicy", key.Name},
		eventTarget{"HTTPRoute", key.Name},
	)
}

// ----------------------------------------------------------------------
// shared helpers

type eventTarget struct {
	kind string
	name string
}

func (r *KubernetesRuntime) runLabelID(ctx context.Context, key types.NamespacedName) (string, error) {
	cr := &mlrunv1alpha1.MLRun{}
	if err := r.ctrl.Get(ctx, key, cr); err != nil {
		return "", err
	}
	id := cr.Labels[mlrunv1alpha1.LabelRunID]
	if id == "" {
		return "", fmt.Errorf("mlrun %s missing %s label", key, mlrunv1alpha1.LabelRunID)
	}
	return id, nil
}

func (r *KubernetesRuntime) serviceLabelID(ctx context.Context, key types.NamespacedName) (string, error) {
	cr := &mlservicev1alpha1.MLService{}
	if err := r.ctrl.Get(ctx, key, cr); err != nil {
		return "", err
	}
	id := cr.Labels[mlservicev1alpha1.LabelServiceID]
	if id == "" {
		return "", fmt.Errorf("mlservice %s missing %s label", key, mlservicev1alpha1.LabelServiceID)
	}
	return id, nil
}

func (r *KubernetesRuntime) listPodsByLabel(ctx context.Context, namespace, labelKey, labelValue string) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	sel := labels.SelectorFromSet(labels.Set{labelKey: labelValue})
	if err := r.ctrl.List(ctx, pods, &client.ListOptions{Namespace: namespace, LabelSelector: sel}); err != nil {
		return nil, err
	}
	return pods, nil
}

// verifyInstance confirms the named Pod exists and carries the workload's id
// label, preventing log / event lookups from leaking Pods that belong to other
// workloads in the same namespace.
func (r *KubernetesRuntime) verifyInstance(ctx context.Context, namespace, instance, labelKey, labelValue string) error {
	var p corev1.Pod
	if err := r.ctrl.Get(ctx, client.ObjectKey{Namespace: namespace, Name: instance}, &p); err != nil {
		return err
	}
	if p.Labels[labelKey] != labelValue {
		return fmt.Errorf("instance %s/%s: %w", namespace, instance, extensions.ErrInstanceNotOwned)
	}
	return nil
}

func (r *KubernetesRuntime) streamLogs(ctx context.Context, namespace, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	if opts == nil {
		opts = &corev1.PodLogOptions{}
	}
	return r.clientset.CoreV1().Pods(namespace).GetLogs(instance, opts).Stream(ctx)
}

func (r *KubernetesRuntime) eventsFor(ctx context.Context, namespace string, targets ...eventTarget) (*eventsv1.EventList, error) {
	var all eventsv1.EventList
	if err := r.ctrl.List(ctx, &all, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}
	out := &eventsv1.EventList{}
	for i := range all.Items {
		e := all.Items[i]
		for _, t := range targets {
			if e.Regarding.Kind == t.kind && e.Regarding.Name == t.name {
				out.Items = append(out.Items, e)
				break
			}
		}
	}
	return out, nil
}

func mergeLabels(dst client.Object, src map[string]string) {
	if len(src) == 0 {
		return
	}
	l := dst.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	for k, v := range src {
		l[k] = v
	}
	dst.SetLabels(l)
}

func ignoreNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
