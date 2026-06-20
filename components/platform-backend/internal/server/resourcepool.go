package server

import "time"

// ResourcePool is a pass-through of cluster-manager's ResourcePool. `units` is
// the embedded spec.units[] array, not a separate REST resource.
type ResourcePool struct {
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	NodeSelector    StringMap      `json:"nodeSelector,omitempty"`
	Tolerations     []Toleration   `json:"tolerations,omitempty"`
	Units           []ResourceUnit `json:"units,omitempty"`
	Labels          StringMap      `json:"labels,omitempty"`
	Annotations     StringMap      `json:"annotations,omitempty"`
	NodeCount       int            `json:"nodeCount,omitempty"`
	ResourceVersion string         `json:"resourceVersion,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt,omitempty"`
}

// ResourcePoolList is a page of ResourcePool.
type ResourcePoolList struct {
	Items         []ResourcePool `json:"items"`
	Count         int            `json:"count" binding:"min=0"`
	ContinueToken string         `json:"continueToken,omitempty"`
}

// ResourcePoolCreateRequest is the body of POST /resource-pools.
type ResourcePoolCreateRequest struct {
	Name         string                      `json:"name" binding:"required,dns1123,min=1,max=40"`
	Description  string                      `json:"description,omitempty" binding:"max=1000"`
	NodeSelector StringMap                   `json:"nodeSelector,omitempty"`
	Tolerations  []Toleration                `json:"tolerations,omitempty"`
	Units        []ResourceUnitCreateRequest `json:"units,omitempty"`
	Labels       StringMap                   `json:"labels,omitempty"`
	Annotations  StringMap                   `json:"annotations,omitempty"`
}

// ResourcePoolPatchRequest patches pool-level fields only.
type ResourcePoolPatchRequest struct {
	Description  string       `json:"description,omitempty" binding:"max=1000"`
	NodeSelector StringMap    `json:"nodeSelector,omitempty"`
	Tolerations  []Toleration `json:"tolerations,omitempty"`
	Labels       StringMap    `json:"labels,omitempty"`
	Annotations  StringMap    `json:"annotations,omitempty"`
}

// ResourceUnit is an embedded entry of ResourcePool.spec.units[].
type ResourceUnit struct {
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Requests     ResourceMap  `json:"requests"`
	Limits       ResourceMap  `json:"limits"`
	NodeSelector StringMap    `json:"nodeSelector,omitempty"`
	Tolerations  []Toleration `json:"tolerations,omitempty"`
	Annotations  StringMap    `json:"annotations,omitempty"`
}

// ResourceUnitList is a page of ResourceUnit.
type ResourceUnitList struct {
	Items         []ResourceUnit `json:"items"`
	Count         int            `json:"count" binding:"min=0"`
	ContinueToken string         `json:"continueToken,omitempty"`
}

// ResourceUnitCreateRequest is the body of POST /resource-pools/{pool}/units.
type ResourceUnitCreateRequest struct {
	Name         string       `json:"name" binding:"required,dns1123,min=1,max=40"`
	Description  string       `json:"description,omitempty" binding:"max=1000"`
	Requests     ResourceMap  `json:"requests" binding:"required"`
	Limits       ResourceMap  `json:"limits" binding:"required"`
	NodeSelector StringMap    `json:"nodeSelector,omitempty"`
	Tolerations  []Toleration `json:"tolerations,omitempty"`
	Annotations  StringMap    `json:"annotations,omitempty"`
}

// ResourceUnitPatchRequest patches a single unit.
type ResourceUnitPatchRequest struct {
	Description  string       `json:"description,omitempty" binding:"max=1000"`
	Requests     ResourceMap  `json:"requests,omitempty"`
	Limits       ResourceMap  `json:"limits,omitempty"`
	NodeSelector StringMap    `json:"nodeSelector,omitempty"`
	Tolerations  []Toleration `json:"tolerations,omitempty"`
	Annotations  StringMap    `json:"annotations,omitempty"`
}
