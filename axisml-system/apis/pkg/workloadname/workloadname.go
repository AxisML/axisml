// Package workloadname defines the deployment-form-neutral workload naming
// contract shared by Compute Service and its runtime implementations.
package workloadname

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AnnotationName carries the physical workload base across runtime
	// adapters. API/DB keys remain logical; the Kubernetes adapter also uses
	// this value as the physical CR metadata.name.
	AnnotationName = "compute.axisml.io/workload-name"
	// AnnotationTenantPrefix records the naming policy used to produce
	// AnnotationName so related resource references can apply the same policy.
	AnnotationTenantPrefix = "compute.axisml.io/workload-tenant-prefix"

	ContainerName = "main"
)

// Base returns the physical workload base for a logical workload.
func Base(tenant, workload string, tenantPrefix bool) string {
	if tenantPrefix && tenant != "" {
		// Include a stable tenant token rather than concatenating two logical
		// names directly. Plain "tenant-workload" is ambiguous: (team-a,
		// hello) and (team, a-hello) both produce team-a-hello. The readable
		// tenant slug plus its hash preserves operator context while making the
		// tuple boundary stable across Kubernetes and Standalone runtimes.
		return dnsName(tenant+"-"+shortHash(tenant)+"-"+workload, 63)
	}
	return dnsName(workload, 63)
}

// Annotate stamps the physical naming decision on a desired workload object.
func Annotate(obj metav1.Object, tenant, workload string, tenantPrefix bool) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[AnnotationName] = Base(tenant, workload, tenantPrefix)
	if tenantPrefix {
		annotations[AnnotationTenantPrefix] = "true"
	} else {
		annotations[AnnotationTenantPrefix] = "false"
	}
	obj.SetAnnotations(annotations)
}

// Workload returns the annotated physical base, falling back to metadata.name
// for CRs created outside Compute Service.
func Workload(obj metav1.Object) string {
	if annotations := obj.GetAnnotations(); annotations != nil {
		if name := annotations[AnnotationName]; name != "" {
			return name
		}
	}
	return obj.GetName()
}

// Role returns the resource base whose generated instances contain the role.
func Role(obj metav1.Object, role string) string {
	workload := dnsName(Workload(obj), 63)
	role = dnsName(role, 40)
	const limit = 57 // reserve "-" plus a five-character runtime suffix
	joined := workload + "-" + role
	if len(joined) <= limit {
		return joined
	}
	hash := shortHash(workload)
	suffix := "-" + hash + "-" + role
	prefixLen := limit - len(suffix)
	if prefixLen < 1 {
		return dnsName(joined, limit)
	}
	return strings.TrimRight(workload[:prefixLen], "-") + suffix
}

// Related returns the physical base for another logical workload in the same
// tenant using the naming policy stamped on obj.
func Related(obj metav1.Object, tenant, logicalName string) string {
	enabled := obj.GetAnnotations()[AnnotationTenantPrefix] == "true"
	return Base(tenant, logicalName, enabled)
}

func dnsName(raw string, limit int) string {
	name := strings.ToLower(raw)
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		valid := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	clean := strings.Trim(b.String(), "-")
	if clean == "" {
		clean = "workload"
	}
	if len(clean) <= limit {
		return clean
	}
	hash := shortHash(raw)
	return strings.TrimRight(clean[:limit-len(hash)-1], "-") + "-" + hash
}

func shortHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:8]
}
