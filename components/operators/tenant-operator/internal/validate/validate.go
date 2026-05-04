// Package validate exposes a pure-function Validate(spec) routine used by
// the controller before any K8s API call. The function is intentionally
// dependency-free so the same code path can later back an admission webhook
// (design §5 / §10).
package validate

import (
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	axisml "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"
)

// Options carries cluster-side configuration that influences validation.
// Currently only the namespace denylist is wired in (design §6.1 risk note).
type Options struct {
	// NamespaceDenylist is the set of Namespace names tenants must not target.
	NamespaceDenylist map[string]struct{}
}

// dns1123LabelRegex is the upstream DNS-1123 label regex (lowercase letters,
// digits, hyphens; must start and end with alphanumeric; max 63 chars). It is
// the same expression k8s.io/apimachinery uses internally for namespace names.
var dns1123LabelRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// ValidateMeta enforces metadata-level invariants that the spec validator
// can't see. The tenant-id label is the orphan-detection anchor used by
// every per-tenant resource label, GC selector, and Compute-side reverse
// lookup (design §3, §6); a tenant without it would silently spawn
// resources keyed on the empty string.
func ValidateMeta(meta *metav1.ObjectMeta) error {
	if meta == nil {
		return fmt.Errorf("metadata is nil")
	}
	if meta.Labels[axisml.LabelTenantID] == "" {
		return fmt.Errorf("metadata.labels[%q] is required", axisml.LabelTenantID)
	}
	return nil
}

// Validate runs all design-mandated structural checks on a Tenant spec.
// It is pure: no K8s calls, no I/O. Returned errors are aggregated into one
// human-readable string suitable for status.message.
func Validate(spec *axisml.TenantSpec, opts Options) error {
	if spec == nil {
		return fmt.Errorf("spec is nil")
	}

	if err := validateNamespace(spec.Namespace, opts); err != nil {
		return err
	}
	if err := validateQuotas(spec.Quotas); err != nil {
		return err
	}
	if err := validateInitResources(spec.InitResources); err != nil {
		return err
	}
	return nil
}

func validateNamespace(ns axisml.NamespaceSpec, opts Options) error {
	if ns.Name == "" {
		return fmt.Errorf("spec.namespace.name is required")
	}
	if len(ns.Name) > 63 {
		return fmt.Errorf("spec.namespace.name %q exceeds 63 characters", ns.Name)
	}
	if !dns1123LabelRegex.MatchString(ns.Name) {
		return fmt.Errorf("spec.namespace.name %q is not a valid DNS-1123 label", ns.Name)
	}
	if _, denied := opts.NamespaceDenylist[ns.Name]; denied {
		return fmt.Errorf("spec.namespace.name %q is in the controller denylist", ns.Name)
	}
	return nil
}

func validateQuotas(quotas []axisml.QuotaSpec) error {
	seen := make(map[string]struct{}, len(quotas))
	for i, q := range quotas {
		if q.Pool == "" {
			return fmt.Errorf("spec.quotas[%d].pool is required", i)
		}
		if q.Name == "" {
			return fmt.Errorf("spec.quotas[%d].name is required", i)
		}
		key := q.Pool + "/" + q.Name
		if _, dup := seen[key]; dup {
			return fmt.Errorf("spec.quotas[%d] duplicates (pool=%s, name=%s)", i, q.Pool, q.Name)
		}
		seen[key] = struct{}{}

		if q.Max == nil {
			return fmt.Errorf("spec.quotas[%d].max is required", i)
		}
		if err := validateMinMax(i, q.Min, q.Max); err != nil {
			return err
		}
	}
	return nil
}

func validateMinMax(idx int, min, max corev1.ResourceList) error {
	// Negativity is checked first so that a negative max produces a
	// max-negative error rather than masquerading as a min>max violation.
	for k, v := range max {
		if v.Sign() < 0 {
			return fmt.Errorf("spec.quotas[%d].max[%s] is negative", idx, k)
		}
	}
	for k, v := range min {
		if v.Sign() < 0 {
			return fmt.Errorf("spec.quotas[%d].min[%s] is negative", idx, k)
		}
		mv, ok := max[k]
		if !ok {
			// min carries a key absent from max; per design §6.2 each min[k]
			// must have a corresponding max[k] >= it, so absence is invalid.
			return fmt.Errorf("spec.quotas[%d].max is missing key %s present in min", idx, k)
		}
		if v.Cmp(mv) > 0 {
			return fmt.Errorf("spec.quotas[%d].min[%s]=%s exceeds max[%s]=%s",
				idx, k, v.String(), k, mv.String())
		}
	}
	return nil
}

func validateInitResources(ir axisml.InitResources) error {
	pullNames, err := uniqueNames("imagePullSecrets", namesOfPullSecrets(ir.ImagePullSecrets))
	if err != nil {
		return err
	}
	secretNames, err := uniqueNames("secrets", namesOfSecrets(ir.Secrets))
	if err != nil {
		return err
	}
	// imagePullSecrets[] and secrets[] both render to the same per-tenant
	// Secret object (PerTenantResourceName collapses both), so a name shared
	// across the two lists would have the reconcilers fight over one Secret.
	for n := range secretNames {
		if _, dup := pullNames[n]; dup {
			return fmt.Errorf(
				"spec.initResources.secrets[].name %q collides with spec.initResources.imagePullSecrets[]: each name must be unique across both lists",
				n,
			)
		}
	}
	if _, err := uniqueNames("configMaps", namesOfConfigMaps(ir.ConfigMaps)); err != nil {
		return err
	}
	if _, err := uniqueNames("serviceAccounts", namesOfServiceAccounts(ir.ServiceAccounts)); err != nil {
		return err
	}

	for i, sa := range ir.ServiceAccounts {
		if sa.Name == "" {
			return fmt.Errorf("spec.initResources.serviceAccounts[%d].name is required", i)
		}
		for j, ref := range sa.ImagePullSecrets {
			if _, ok := pullNames[ref]; !ok {
				return fmt.Errorf(
					"spec.initResources.serviceAccounts[%d].imagePullSecrets[%d]=%q is not declared in spec.initResources.imagePullSecrets[]",
					i, j, ref,
				)
			}
		}
		if sa.RBAC != nil && sa.RBAC.RoleRef != nil {
			switch sa.RBAC.RoleRef.Kind {
			case "Role", "ClusterRole":
			default:
				return fmt.Errorf(
					"spec.initResources.serviceAccounts[%d].rbac.roleRef.kind %q must be Role or ClusterRole",
					i, sa.RBAC.RoleRef.Kind,
				)
			}
			if sa.RBAC.RoleRef.Name == "" {
				return fmt.Errorf("spec.initResources.serviceAccounts[%d].rbac.roleRef.name is required when roleRef is set", i)
			}
		}
	}

	return nil
}

func uniqueNames(field string, names []string) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(names))
	for i, n := range names {
		if n == "" {
			return nil, fmt.Errorf("spec.initResources.%s[%d].name is required", field, i)
		}
		if _, dup := seen[n]; dup {
			return nil, fmt.Errorf("spec.initResources.%s[%d].name %q is duplicated", field, i, n)
		}
		seen[n] = struct{}{}
	}
	return seen, nil
}

func namesOfPullSecrets(in []axisml.ImagePullSecretSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}

func namesOfSecrets(in []axisml.SecretSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}

func namesOfConfigMaps(in []axisml.ConfigMapSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}

func namesOfServiceAccounts(in []axisml.ServiceAccountSpec) []string {
	out := make([]string, len(in))
	for i, x := range in {
		out[i] = x.Name
	}
	return out
}

// DefaultNamespaceDenylist returns the set design §6.1 risk note recommends
// to start with. Helm values can override or extend it via the
// NAMESPACE_DENYLIST env var (see internal/config).
func DefaultNamespaceDenylist() map[string]struct{} {
	return map[string]struct{}{
		"kube-system":     {},
		"kube-public":     {},
		"kube-node-lease": {},
		"default":         {},
		"axisml-system":   {},
		"axisml-infra":    {},
	}
}
