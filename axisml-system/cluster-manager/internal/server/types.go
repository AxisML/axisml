// Package server defines request/response types and shared error helpers
// for the cluster-manager REST API. The service is a stateless shell over
// the ResourcePool CRD (cluster-scoped, axisml.io/v1alpha1) plus the
// embedded `spec.units[]` array; types here mirror the OpenAPI contract
// in docs/system_design/apis/cluster-manager.yaml.
package server

import (
	"net/http"
	"time"
	"unicode"

	corev1 "k8s.io/api/core/v1"

	"github.com/gin-gonic/gin"
)

// ResourcePool matches the OpenAPI ResourcePool schema. It wraps the
// underlying CR's metadata.{name,labels,annotations,resourceVersion} and
// spec fields. No separate id — CR name is the stable handle.
type ResourcePool struct {
	Name            string              `json:"name" desc:"Pool name; the stable, immutable handle (CR metadata.name)."`
	Description     string              `json:"description,omitempty" desc:"Human-readable description of the pool."`
	NodeSelector    map[string]string   `json:"nodeSelector,omitempty" desc:"Node labels that workloads scheduled into this pool must match."`
	Tolerations     []corev1.Toleration `json:"tolerations,omitempty" desc:"Tolerations applied to workloads scheduled into this pool."`
	Units           []ResourceUnit      `json:"units" desc:"Resource units (allocatable shapes) offered by this pool."`
	Labels          map[string]string   `json:"labels,omitempty" desc:"User-defined labels on the pool."`
	Annotations     map[string]string   `json:"annotations,omitempty" desc:"User-defined annotations on the pool."`
	ResourceVersion string              `json:"resourceVersion,omitempty" desc:"Opaque CR resourceVersion for optimistic concurrency."`
	CreatedAt       time.Time           `json:"createdAt" desc:"Pool creation timestamp (RFC3339)."`
	UpdatedAt       time.Time           `json:"updatedAt,omitempty" desc:"Last modification timestamp (RFC3339)."`
}

// ResourceUnit is one entry of pool.spec.units[]. Identified by the
// (poolName, name) tuple in the URL — no independent CR.
type ResourceUnit struct {
	Name         string              `json:"name" desc:"Unit name; unique within the pool and immutable."`
	Description  string              `json:"description,omitempty" desc:"Human-readable description of the unit."`
	Requests     corev1.ResourceList `json:"requests" desc:"Resource requests granted per quantity of this unit (e.g. cpu, memory, nvidia.com/gpu)."`
	Limits       corev1.ResourceList `json:"limits" desc:"Resource limits per quantity of this unit."`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty" desc:"Node labels workloads using this unit must match (overrides the pool selector)."`
	Annotations  map[string]string   `json:"annotations,omitempty" desc:"User-defined annotations on the unit."`
}

// CreateResourcePoolRequest is the body for POST /api/v1/resourcepools.
type CreateResourcePoolRequest struct {
	Name         string                      `json:"name" desc:"Pool name to create; must be unique and DNS-1123 compliant."`
	Description  string                      `json:"description,omitempty" desc:"Human-readable description of the pool."`
	NodeSelector map[string]string           `json:"nodeSelector,omitempty" desc:"Node labels that workloads scheduled into this pool must match."`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty" desc:"Tolerations applied to workloads scheduled into this pool."`
	Units        []CreateResourceUnitRequest `json:"units,omitempty" desc:"Initial set of resource units to create inline with the pool."`
	Labels       map[string]string           `json:"labels,omitempty" desc:"User-defined labels to set on the pool."`
	Annotations  map[string]string           `json:"annotations,omitempty" desc:"User-defined annotations to set on the pool."`
}

// PatchResourcePoolRequest covers the pool-level mutable fields. `name`
// is immutable; use unit sub-routes for unit-level mutations.
type PatchResourcePoolRequest struct {
	Description  *string             `json:"description,omitempty" desc:"New description; omit to leave unchanged, empty string to clear."`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty" desc:"Replacement node selector for the pool."`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty" desc:"Replacement tolerations for the pool."`
	Labels       map[string]string   `json:"labels,omitempty" desc:"Replacement labels for the pool."`
	Annotations  map[string]string   `json:"annotations,omitempty" desc:"Replacement annotations for the pool."`
}

// CreateResourceUnitRequest is the body for POST .../units.
type CreateResourceUnitRequest struct {
	Name         string              `json:"name" desc:"Unit name to create; unique within the pool."`
	Description  string              `json:"description,omitempty" desc:"Human-readable description of the unit."`
	Requests     corev1.ResourceList `json:"requests" desc:"Resource requests granted per quantity of this unit."`
	Limits       corev1.ResourceList `json:"limits" desc:"Resource limits per quantity of this unit."`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty" desc:"Node labels workloads using this unit must match."`
	Annotations  map[string]string   `json:"annotations,omitempty" desc:"User-defined annotations to set on the unit."`
}

// PatchResourceUnitRequest covers the unit-level mutable fields. `name`
// is immutable.
type PatchResourceUnitRequest struct {
	Description  *string             `json:"description,omitempty" desc:"New description; omit to leave unchanged, empty string to clear."`
	Requests     corev1.ResourceList `json:"requests,omitempty" desc:"Replacement resource requests for the unit."`
	Limits       corev1.ResourceList `json:"limits,omitempty" desc:"Replacement resource limits for the unit."`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty" desc:"Replacement node selector for the unit."`
	Annotations  map[string]string   `json:"annotations,omitempty" desc:"Replacement annotations for the unit."`
}

// ResourcePoolList is the LIST response.
type ResourcePoolList struct {
	Items         []ResourcePool `json:"items" desc:"Page of resource pools."`
	Count         int            `json:"count" desc:"Number of pools in this page."`
	ContinueToken string         `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page; empty when no more pages."`
}

// ResourceUnitList is the LIST response for units inside one pool.
type ResourceUnitList struct {
	Items []ResourceUnit `json:"items" desc:"Resource units in the pool."`
	Count int            `json:"count" desc:"Number of units returned."`
}

// Error mirrors RFC 7807 application/problem+json.
type Error struct {
	Type   string `json:"type" desc:"URI reference identifying the problem type (RFC 7807); about:blank when unspecified."`
	Title  string `json:"title" desc:"Short, human-readable summary of the problem."`
	Status int    `json:"status" desc:"HTTP status code for this occurrence of the problem."`
	Detail string `json:"detail,omitempty" desc:"Human-readable explanation specific to this occurrence."`
	Code   string `json:"code,omitempty" desc:"Stable machine-readable error code for programmatic handling."`
}

// AbortWithProblem writes an RFC 7807 problem response and stops the gin chain.
func AbortWithProblem(c *gin.Context, status int, code, title, detail string) {
	c.AbortWithStatusJSON(status, Error{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	})
}

// HealthStatus is the body returned by /healthz, /readyz on success.
type HealthStatus struct {
	Status string `json:"status" desc:"Probe status string (e.g. ok)."`
}

// LastModifiedByAnnotation tracks the X-Axisml-User that performed the
// most recent mutation (audit hint; the K8s API also persists this).
const LastModifiedByAnnotation = "axisml.io/last-modified-by"

// DescriptionAnnotation surfaces the API type's `description` field through
// the CR's metadata.annotations[axisml.io/description] (no dedicated CR
// field — keeps the API admin-friendly without polluting spec).
const DescriptionAnnotation = "axisml.io/description"

// HeaderUser is the request header carrying the calling end-user.
const HeaderUser = "X-Axisml-User"

// MaxUserHeaderLen caps the X-Axisml-User length so a malicious caller
// cannot bloat metadata.annotations (K8s caps annotations at 256 KiB
// total). 253 matches the DNS-1123 subdomain bound, which is the largest
// caller identity we currently expect (email, k8s username).
const MaxUserHeaderLen = 253

// RequireUser is gin middleware that gates /api/v1 on a valid
// X-Axisml-User header. It only audits the caller; the gateway is the
// real authn boundary (see axisml-system/deploy/helm/templates/
// cluster-manager/networkpolicy.yaml — only the gateway namespace may
// reach :8080). Without that NetworkPolicy any pod in the cluster could
// set the header and impersonate.
func RequireUser(c *gin.Context) {
	user := c.GetHeader(HeaderUser)
	if user == "" {
		AbortWithProblem(c, http.StatusUnauthorized, "MissingUser",
			"X-Axisml-User header required", "")
		return
	}
	if err := validateUserHeader(user); err != nil {
		AbortWithProblem(c, http.StatusBadRequest, "InvalidUser",
			"X-Axisml-User header malformed", err.Error())
		return
	}
	c.Next()
}

// validateUserHeader rejects control chars, whitespace, and overly long
// values before the header lands in metadata.annotations.
func validateUserHeader(v string) error {
	if len(v) > MaxUserHeaderLen {
		return errUserTooLong
	}
	for _, r := range v {
		if r == ' ' || unicode.IsControl(r) || unicode.IsSpace(r) {
			return errUserBadChars
		}
	}
	return nil
}

var (
	errUserTooLong  = &headerError{msg: "value exceeds 253 chars"}
	errUserBadChars = &headerError{msg: "value contains whitespace or control characters"}
)

type headerError struct{ msg string }

func (e *headerError) Error() string { return e.msg }
