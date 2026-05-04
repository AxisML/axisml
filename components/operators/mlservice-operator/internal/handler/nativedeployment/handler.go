// Package nativedeployment implements the (native, deployment) Handler:
// Deployment + ClusterIP Service, with optional HTTPRoute when spec.route is
// enabled. See mlservice-operator.md §8.1.
package nativedeployment

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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	axisml "github.com/axisml/axisml/components/operators/mlservice-operator/api/v1alpha1"
	"github.com/axisml/axisml/components/operators/mlservice-operator/internal/handler"
)

const (
	// GatewayParentName is the AxisML Envoy Gateway listener that all
	// MLService HTTPRoutes attach to. Cross-namespace attachment is governed
	// by Gateway.spec.listeners.allowedRoutes.namespaces.from=All on the
	// shared Gateway (provisioned by axisml-infra; see infra.md §3.1) — NOT
	// by ReferenceGrant. ReferenceGrant only matters for cross-namespace
	// BackendRef, and the HTTPRoute + Service emitted here live in the same
	// (tenant) namespace.
	GatewayParentName      = "axisml-gateway"
	GatewayParentNamespace = "axisml-infra"
)

// Handler is the (native, deployment) implementation.
type Handler struct {
	client client.Client
}

// New is the registry Factory entry point.
func New(mgr manager.Manager) (handler.Handler, error) {
	return &Handler{client: mgr.GetClient()}, nil
}

func init() {
	handler.Register(handler.Key{Backend: "native", Engine: "deployment"}, New)
}

func (h *Handler) Key() handler.Key {
	return handler.Key{Backend: "native", Engine: "deployment"}
}

// Validate enforces §8.1 "single role named predictor" and the §3.3 / §6
// constraints we can check from spec alone (route shape, ports, schedule).
func (h *Handler) Validate(spec *axisml.MLServiceSpec) handler.Validation {
	v := handler.Validation{}
	if len(spec.Roles) != 1 {
		v.Errors = append(v.Errors,
			fmt.Sprintf("(native, deployment) requires exactly one role; got %d", len(spec.Roles)))
		return v
	}
	role := spec.Roles[0]
	if role.Name != axisml.DefaultRoleName {
		v.Errors = append(v.Errors,
			fmt.Sprintf("(native, deployment) role must be named %q; got %q",
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
	if spec.ModelRef.Name == "" || spec.ModelRef.Version == "" {
		v.Errors = append(v.Errors, "modelRef.name and modelRef.version are required")
	}
	if spec.Scheduling.Quota == "" {
		v.Errors = append(v.Errors, "scheduling.quota is required")
	}
	if spec.Backend.Config != nil && len(spec.Backend.Config.Raw) > 0 {
		v.Warnings = append(v.Warnings,
			"(native, deployment) ignores backend.config; reserved for future fields")
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

// Reconcile creates or updates the Deployment, the ClusterIP Service, and
// (when route.enabled=true) the HTTPRoute. It is idempotent and does not
// rebuild Pods on roles[0].replicas-only changes.
func (h *Handler) Reconcile(ctx context.Context, mls *axisml.MLService) (handler.Result, error) {
	dep := buildDeployment(mls)
	if err := h.upsertDeployment(ctx, mls, dep); err != nil {
		return handler.Result{}, fmt.Errorf("upsert deployment: %w", err)
	}
	svc := buildService(mls)
	if err := h.upsertService(ctx, mls, svc); err != nil {
		return handler.Result{}, fmt.Errorf("upsert service: %w", err)
	}
	if mls.Spec.Route != nil && mls.Spec.Route.Enabled {
		route := buildHTTPRoute(mls)
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

// Cleanup is a no-op: ownerReference cascade deletes the Deployment, Service
// and HTTPRoute when MLService disappears (§6 / §10).
func (h *Handler) Cleanup(_ context.Context, _ *axisml.MLService) error {
	return nil
}

func (h *Handler) WatchTargets() []client.Object {
	return []client.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
		&gwapiv1.HTTPRoute{},
	}
}

// RequiredRBAC mirrors §8.1's RBAC table. SecurityPolicy / BackendTrafficPolicy
// rules are intentionally omitted until those resources are actually emitted.
func (h *Handler) RequiredRBAC() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{APIGroups: []string{"apps"}, Resources: []string{"deployments"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"services", "pods"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
		{APIGroups: []string{""}, Resources: []string{"events"},
			Verbs: []string{"create", "patch"}},
		{APIGroups: []string{"gateway.networking.k8s.io"}, Resources: []string{"httproutes"},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch", "delete"}},
	}
}

// appliedSpecHashAnnotation records the SHA-256 of the rendered desired
// Deployment spec the operator last wrote. It lets reconciles short-circuit
// when nothing the operator owns has changed — even after the apiserver
// fills in defaulted fields (terminationMessagePath, dnsPolicy, etc.) on the
// returned object. Without this, comparing rendered (undefaulted) desired
// against fetched (defaulted) current returns false-mismatch on every
// reconcile, generating a no-op patch loop that churns metadata.generation
// and starves the Deployment controller's availability accounting.
const appliedSpecHashAnnotation = "axisml.io/applied-spec-hash"

// upsertDeployment is a focused create-or-patch. We intentionally avoid the
// stock CreateOrUpdate/CreateOrPatch helpers: they invoke the mutate function
// before reading the existing object, which makes label-merge semantics
// awkward. Doing it inline keeps the Pod template injection deterministic.
func (h *Handler) upsertDeployment(ctx context.Context, mls *axisml.MLService, desired *appsv1.Deployment) error {
	if err := controllerutil.SetControllerReference(mls, desired, h.client.Scheme()); err != nil {
		return err
	}
	hash, err := hashDeploymentSpec(desired)
	if err != nil {
		return fmt.Errorf("hash desired spec: %w", err)
	}
	if desired.Annotations == nil {
		desired.Annotations = map[string]string{}
	}
	desired.Annotations[appliedSpecHashAnnotation] = hash

	current := &appsv1.Deployment{}
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

// hashDeploymentSpec hashes the rendered fields the operator owns: spec
// (replicas / selector / template) plus the labels we stamp. The annotation
// itself is excluded — it is set after the hash is computed.
func hashDeploymentSpec(d *appsv1.Deployment) (string, error) {
	subset := struct {
		Spec   appsv1.DeploymentSpec `json:"spec"`
		Labels map[string]string     `json:"labels"`
	}{
		Spec:   d.Spec,
		Labels: d.Labels,
	}
	raw, err := json.Marshal(subset)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (h *Handler) upsertService(ctx context.Context, mls *axisml.MLService, desired *corev1.Service) error {
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
	patched := current.DeepCopy()
	// Service.spec.clusterIP is immutable: preserve whatever the apiserver
	// already assigned. Everything else is overwritten from desired.
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

// mergeLabels lets the operator add/overwrite axisml-owned labels without
// stripping labels other actors (sidecar injectors, mesh control planes) may
// have placed on the resource.
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
