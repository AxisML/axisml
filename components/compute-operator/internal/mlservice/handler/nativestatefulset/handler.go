// Package nativestatefulset implements the (native, statefulset) Handler:
// StatefulSet + headless Service, with optional HTTPRoute when spec.route is
// enabled. See mlservice-operator.md §6.6.2.
package nativestatefulset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
	"github.com/axisml/axisml/components/compute-operator/internal/mlservice/handler"
)

const (
	// GatewayParentName / GatewayParentNamespace mirror nativedeployment: the
	// shared Envoy Gateway listener provisioned by axisml-infra. Cross-namespace
	// attachment is governed by the Gateway's allowedRoutes.namespaces.from=All.
	GatewayParentName      = "axisml-gateway"
	GatewayParentNamespace = "axisml-infra"
)

// handlerKey is the single source of truth for the (backend, engine) tuple
// this handler claims. init() and Key() both reference it so a typo in one
// can't desynchronise the registry from the dispatcher route.
var handlerKey = handler.Key{Backend: "native", Engine: "statefulset"}

// Handler is the (native, statefulset) implementation.
type Handler struct {
	client client.Client
}

// New is the registry Factory entry point.
func New(mgr manager.Manager) (handler.Handler, error) {
	return &Handler{client: mgr.GetClient()}, nil
}

func init() {
	handler.Register(handlerKey, New)
}

func (h *Handler) Key() handler.Key {
	return handlerKey
}

// Validate enforces §6.6.2 "single role named predictor" and the §3.4 / §6
// constraints we can check from spec alone. backend.config is strict-decoded
// (unknown keys rejected) so deferred fields like volumeClaimTemplates and
// updateStrategy cannot slip in unimplemented.
func (h *Handler) Validate(spec *axisml.MLServiceSpec) handler.Validation {
	v := handler.Validation{}
	if len(spec.Roles) != 1 {
		v.Errors = append(v.Errors,
			fmt.Sprintf("(native, statefulset) requires exactly one role; got %d", len(spec.Roles)))
		return v
	}
	role := spec.Roles[0]
	if role.Name != axisml.DefaultRoleName {
		v.Errors = append(v.Errors,
			fmt.Sprintf("(native, statefulset) role must be named %q; got %q",
				axisml.DefaultRoleName, role.Name))
	}
	if role.Replicas < 0 {
		v.Errors = append(v.Errors,
			fmt.Sprintf("roles[0].replicas must be >=0; got %d", role.Replicas))
	}
	if role.Template.Image == "" {
		v.Errors = append(v.Errors, "roles[0].template.image is required")
	}
	if len(role.Template.Ports) == 0 {
		v.Errors = append(v.Errors, "roles[0].template.ports must declare at least one port")
	}
	seenPort := map[string]bool{}
	for i, p := range role.Template.Ports {
		if p.Name == "" {
			v.Errors = append(v.Errors,
				fmt.Sprintf("roles[0].template.ports[%d].name is required", i))
		}
		if seenPort[p.Name] {
			v.Errors = append(v.Errors,
				fmt.Sprintf("roles[0].template.ports[%d]: duplicate name %q", i, p.Name))
		}
		seenPort[p.Name] = true
		if p.ContainerPort <= 0 {
			v.Errors = append(v.Errors,
				fmt.Sprintf("roles[0].template.ports[%d].containerPort must be >0", i))
		}
	}
	if spec.Scheduling.Quota == "" {
		v.Errors = append(v.Errors, "scheduling.quota is required")
	}
	if _, err := parseBackendConfig(spec.Backend.Config); err != nil {
		v.Errors = append(v.Errors, fmt.Sprintf("backend.config invalid: %s", err.Error()))
	}
	if spec.RunPolicy.ProgressDeadlineSeconds != nil {
		v.Warnings = append(v.Warnings,
			"runPolicy.progressDeadlineSeconds has no StatefulSet equivalent; ignored")
	}
	if spec.Route != nil && spec.Route.Enabled {
		if len(role.Template.Ports) > 1 && spec.Route.PortName == "" {
			v.Errors = append(v.Errors,
				"route.portName must be set when the role exposes multiple ports")
		}
		if spec.Route.PortName != "" && !seenPort[spec.Route.PortName] {
			v.Errors = append(v.Errors,
				fmt.Sprintf("route.portName %q does not match any role port", spec.Route.PortName))
		}
		if spec.Route.TargetRole != "" && spec.Route.TargetRole != role.Name {
			v.Errors = append(v.Errors,
				fmt.Sprintf("route.targetRole %q does not match the only role %q",
					spec.Route.TargetRole, role.Name))
		}
		if spec.Route.Auth != nil && spec.Route.Auth.Type != "" && spec.Route.Auth.Type != axisml.RouteAuthNone {
			v.Warnings = append(v.Warnings,
				"route.auth is not yet wired; SecurityPolicy creation deferred to a follow-up")
		}
		if spec.Route.RateLimit != nil || spec.Route.Timeout != "" {
			v.Warnings = append(v.Warnings,
				"route.rateLimit/timeout are not yet wired; BackendTrafficPolicy creation deferred to a follow-up")
		}
	}
	return v
}

// Reconcile creates or updates the StatefulSet, the headless Service, and
// (when route.enabled=true) the HTTPRoute. It is idempotent and does not
// rebuild Pods on roles[0].replicas-only changes.
func (h *Handler) Reconcile(ctx context.Context, mls *axisml.MLService) (handler.Result, error) {
	cfg, err := parseBackendConfig(mls.Spec.Backend.Config)
	if err != nil {
		return handler.Result{}, fmt.Errorf("parse backend.config: %w", err)
	}
	sts := buildStatefulSet(mls, cfg)
	if err := h.upsertStatefulSet(ctx, mls, sts); err != nil {
		return handler.Result{}, fmt.Errorf("upsert statefulset: %w", err)
	}
	svc := buildHeadlessService(mls, cfg)
	if err := h.upsertHeadlessService(ctx, mls, svc); err != nil {
		return handler.Result{}, fmt.Errorf("upsert headless service: %w", err)
	}
	if mls.Spec.Route != nil && mls.Spec.Route.Enabled {
		route := buildHTTPRoute(mls, defaultedServiceName(mls, cfg))
		if err := h.upsertHTTPRoute(ctx, mls, route); err != nil {
			return handler.Result{}, fmt.Errorf("upsert httproute: %w", err)
		}
	}
	return handler.Result{}, nil
}

// MapStatus is pure: dispatcher passes the cached children, this function
// computes the StatusUpdate without touching the API server.
func (h *Handler) MapStatus(snap handler.Snapshot) handler.StatusUpdate {
	return mapStatus(snap)
}

// Cleanup is a no-op: ownerReference cascade deletes the StatefulSet, Service
// and HTTPRoute when MLService disappears.
func (h *Handler) Cleanup(_ context.Context, _ *axisml.MLService) error {
	return nil
}

func (h *Handler) WatchTargets() []client.Object {
	return []client.Object{
		&appsv1.StatefulSet{},
		&corev1.Service{},
		&gwapiv1.HTTPRoute{},
	}
}

// RequiredRBAC mirrors §6.6.2's RBAC table. SecurityPolicy / BackendTrafficPolicy
// rules are intentionally omitted until those resources are actually emitted.
func (h *Handler) RequiredRBAC() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{APIGroups: []string{"apps"}, Resources: []string{"statefulsets"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"services", "pods"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"events"},
			Verbs: []string{"create", "patch"}},
		{APIGroups: []string{"gateway.networking.k8s.io"}, Resources: []string{"httproutes"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
	}
}

// appliedSpecHashAnnotation records the SHA-256 of the rendered desired spec;
// see nativedeployment for the rationale (skip no-op patches when apiserver
// defaulting otherwise causes a false-mismatch on every reconcile).
const appliedSpecHashAnnotation = "axisml.io/applied-spec-hash"

func (h *Handler) upsertStatefulSet(ctx context.Context, mls *axisml.MLService, desired *appsv1.StatefulSet) error {
	if err := controllerutil.SetControllerReference(mls, desired, h.client.Scheme()); err != nil {
		return err
	}
	hash, err := hashStatefulSetSpec(desired)
	if err != nil {
		return fmt.Errorf("hash desired spec: %w", err)
	}
	if desired.Annotations == nil {
		desired.Annotations = map[string]string{}
	}
	desired.Annotations[appliedSpecHashAnnotation] = hash

	current := &appsv1.StatefulSet{}
	getErr := h.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case apierrors.IsNotFound(getErr):
		return h.client.Create(ctx, desired)
	case getErr != nil:
		return getErr
	}

	if current.Annotations[appliedSpecHashAnnotation] == hash &&
		equality.Semantic.DeepEqual(current.Labels, mergeLabels(current.Labels, desired.Labels)) {
		return nil
	}

	patched := current.DeepCopy()
	patched.Labels = mergeLabels(current.Labels, desired.Labels)
	if patched.Annotations == nil {
		patched.Annotations = map[string]string{}
	}
	for k, v := range desired.Annotations {
		patched.Annotations[k] = v
	}
	patched.OwnerReferences = desired.OwnerReferences
	patched.Spec = desired.Spec
	return h.client.Patch(ctx, patched, client.MergeFrom(current))
}

func hashStatefulSetSpec(s *appsv1.StatefulSet) (string, error) {
	subset := struct {
		Spec   appsv1.StatefulSetSpec `json:"spec"`
		Labels map[string]string      `json:"labels"`
	}{
		Spec:   s.Spec,
		Labels: s.Labels,
	}
	raw, err := json.Marshal(subset)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// upsertHeadlessService refuses to adopt a Service that exists but is owned
// by something else (or no controller at all). Adoption would silently
// transfer ownership and risk accidental data exposure when a user picks a
// serviceName that collides with an unrelated workload.
func (h *Handler) upsertHeadlessService(ctx context.Context, mls *axisml.MLService, desired *corev1.Service) error {
	if err := controllerutil.SetControllerReference(mls, desired, h.client.Scheme()); err != nil {
		return err
	}
	current := &corev1.Service{}
	err := h.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case apierrors.IsNotFound(err):
		return h.client.Create(ctx, desired)
	case err != nil:
		return err
	}
	if owner := metav1.GetControllerOf(current); owner == nil || owner.UID != mls.UID {
		return fmt.Errorf("service %s/%s already exists and is not owned by this MLService",
			desired.Namespace, desired.Name)
	}
	patched := current.DeepCopy()
	// Service.spec.clusterIP is immutable: preserve whatever the apiserver
	// already assigned. For headless services the value is "None" and round-
	// trips trivially; the same code path is correct for both shapes.
	clusterIP := patched.Spec.ClusterIP
	clusterIPs := patched.Spec.ClusterIPs
	patched.Spec = desired.Spec
	patched.Spec.ClusterIP = clusterIP
	patched.Spec.ClusterIPs = clusterIPs
	patched.Labels = mergeLabels(current.Labels, desired.Labels)
	if equality.Semantic.DeepEqual(current.Spec, patched.Spec) &&
		equality.Semantic.DeepEqual(current.Labels, patched.Labels) {
		return nil
	}
	return h.client.Patch(ctx, patched, client.MergeFrom(current))
}

func (h *Handler) upsertHTTPRoute(ctx context.Context, mls *axisml.MLService, desired *gwapiv1.HTTPRoute) error {
	if err := controllerutil.SetControllerReference(mls, desired, h.client.Scheme()); err != nil {
		return err
	}
	current := &gwapiv1.HTTPRoute{}
	err := h.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, current)
	switch {
	case apierrors.IsNotFound(err):
		return h.client.Create(ctx, desired)
	case err != nil:
		return err
	}
	patched := current.DeepCopy()
	patched.Spec = desired.Spec
	patched.Labels = mergeLabels(current.Labels, desired.Labels)
	if equality.Semantic.DeepEqual(current.Spec, patched.Spec) &&
		equality.Semantic.DeepEqual(current.Labels, patched.Labels) {
		return nil
	}
	return h.client.Patch(ctx, patched, client.MergeFrom(current))
}

func mergeLabels(existing, desired map[string]string) map[string]string {
	if len(existing) == 0 {
		return desired
	}
	out := make(map[string]string, len(existing)+len(desired))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range desired {
		out[k] = v
	}
	return out
}
