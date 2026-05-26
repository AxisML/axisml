// Package server defines request/response DTOs and shared error helpers
// for the cluster-manager REST API. The service is a stateless shell over
// the ResourcePool CRD (cluster-scoped, axisml.io/v1alpha1) plus the
// embedded `spec.units[]` array; types here mirror the OpenAPI contract
// in docs/system_design/apis/cluster-manager.yaml.
package server

import (
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/gin-gonic/gin"
)

// ResourcePoolDTO matches the OpenAPI ResourcePool schema. It wraps the
// underlying CR's metadata.{name,labels,annotations,resourceVersion} and
// spec fields. No separate id — CR name is the stable handle.
type ResourcePoolDTO struct {
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	NodeSelector    map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations     []corev1.Toleration `json:"tolerations,omitempty"`
	Units           []ResourceUnitDTO   `json:"units"`
	Labels          map[string]string   `json:"labels,omitempty"`
	Annotations     map[string]string   `json:"annotations,omitempty"`
	ResourceVersion string              `json:"resourceVersion,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt,omitempty"`
}

// ResourceUnitDTO is one entry of pool.spec.units[]. Identified by the
// (poolName, name) tuple in the URL — no independent CR.
type ResourceUnitDTO struct {
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Requests     corev1.ResourceList `json:"requests"`
	Limits       corev1.ResourceList `json:"limits"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// CreateResourcePoolRequest is the body for POST /api/v1/resource-pools.
type CreateResourcePoolRequest struct {
	Name         string                      `json:"name"`
	Description  string                      `json:"description,omitempty"`
	NodeSelector map[string]string           `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty"`
	Units        []CreateResourceUnitRequest `json:"units,omitempty"`
	Labels       map[string]string           `json:"labels,omitempty"`
	Annotations  map[string]string           `json:"annotations,omitempty"`
}

// PatchResourcePoolRequest covers the pool-level mutable fields. `name`
// is immutable; use unit sub-routes for unit-level mutations.
type PatchResourcePoolRequest struct {
	Description  *string             `json:"description,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// CreateResourceUnitRequest is the body for POST .../resource-units.
type CreateResourceUnitRequest struct {
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Requests     corev1.ResourceList `json:"requests"`
	Limits       corev1.ResourceList `json:"limits"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// PatchResourceUnitRequest covers the unit-level mutable fields. `name`
// is immutable.
type PatchResourceUnitRequest struct {
	Description  *string             `json:"description,omitempty"`
	Requests     corev1.ResourceList `json:"requests,omitempty"`
	Limits       corev1.ResourceList `json:"limits,omitempty"`
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// ResourcePoolList is the LIST response.
type ResourcePoolList struct {
	Items         []ResourcePoolDTO `json:"items"`
	Count         int               `json:"count"`
	ContinueToken string            `json:"continueToken,omitempty"`
}

// ResourceUnitList is the LIST response for units inside one pool.
type ResourceUnitList struct {
	Items []ResourceUnitDTO `json:"items"`
	Count int               `json:"count"`
}

// Problem mirrors RFC 7807 application/problem+json.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Code   string `json:"code,omitempty"`
}

// AbortWithProblem writes an RFC 7807 problem response and stops the gin chain.
func AbortWithProblem(c *gin.Context, status int, code, title, detail string) {
	c.AbortWithStatusJSON(status, Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	})
}

// HealthStatus is the body returned by /healthz, /readyz on success.
type HealthStatus struct {
	Status string `json:"status"`
}

// LastModifiedByAnnotation tracks the X-Axisml-User that performed the
// most recent mutation (audit hint; the K8s API also persists this).
const LastModifiedByAnnotation = "axisml.io/last-modified-by"

// DescriptionAnnotation surfaces the DTO's `description` field through
// the CR's metadata.annotations[axisml.io/description] (no dedicated CR
// field — keeps the API admin-friendly without polluting spec).
const DescriptionAnnotation = "axisml.io/description"

// HeaderUser is the request header carrying the calling end-user.
const HeaderUser = "X-Axisml-User"

// RequireUser is gin middleware that 401s if X-Axisml-User is missing on
// any /api/v1 request.
func RequireUser(c *gin.Context) {
	if c.GetHeader(HeaderUser) == "" {
		AbortWithProblem(c, http.StatusUnauthorized, "MissingUser",
			"X-Axisml-User header required", "")
		return
	}
	c.Next()
}
