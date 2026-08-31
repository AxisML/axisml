package server

import "time"

// ResourcePool is a pass-through of cluster-manager's ResourcePool. `units` is
// the embedded spec.units[] array, not a separate REST resource.
type ResourcePool struct {
	Name            string         `json:"name" desc:"Cluster-scoped resource pool name (unique across the cluster)."`
	Description     string         `json:"description,omitempty" desc:"Free-text pool description."`
	NodeSelector    StringMap      `json:"nodeSelector,omitempty" desc:"Node labels selecting the nodes that back this pool."`
	Capacity        ResourceMap    `json:"capacity,omitempty" desc:"Optional capacity override; omitted means capacity is derived from matching runtime nodes."`
	Units           []ResourceUnit `json:"units,omitempty" desc:"Resource unit shapes embedded in the pool's spec.units[]."`
	Labels          StringMap      `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations     StringMap      `json:"annotations,omitempty" desc:"User-defined annotations."`
	NodeCount       int            `json:"nodeCount,omitempty" desc:"Number of nodes currently matched by the pool's node selector (read-only)."`
	ResourceVersion string         `json:"resourceVersion,omitempty" desc:"Kubernetes resourceVersion for optimistic concurrency."`
	CreatedAt       time.Time      `json:"createdAt" desc:"Time the pool was created."`
	UpdatedAt       time.Time      `json:"updatedAt,omitempty" desc:"Time the pool was last updated."`
}

// ResourcePoolList is a page of ResourcePool.
type ResourcePoolList struct {
	Items         []ResourcePool `json:"items" desc:"Resource pools in this page."`
	Count         int            `json:"count" binding:"min=0" desc:"Number of pools in this page."`
	ContinueToken string         `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
}

// ResourcePoolCreateRequest is the body of POST /resource-pools.
type ResourcePoolCreateRequest struct {
	Name         string                      `json:"name" binding:"required,dns1123,min=1,max=40" desc:"Cluster-scoped resource pool name (unique across the cluster)."`
	Description  string                      `json:"description,omitempty" binding:"max=1000" desc:"Free-text pool description."`
	NodeSelector StringMap                   `json:"nodeSelector,omitempty" desc:"Node labels selecting the nodes that back this pool."`
	Capacity     ResourceMap                 `json:"capacity,omitempty" desc:"Optional capacity override; omit to derive capacity from matching runtime nodes."`
	Units        []ResourceUnitCreateRequest `json:"units,omitempty" desc:"Resource unit shapes to embed in the pool."`
	Labels       StringMap                   `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations  StringMap                   `json:"annotations,omitempty" desc:"User-defined annotations."`
}

// ResourcePoolPatchRequest patches pool-level fields only.
type ResourcePoolPatchRequest struct {
	Description  string      `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text pool description."`
	NodeSelector StringMap   `json:"nodeSelector,omitempty" desc:"Replacement node selector."`
	Capacity     ResourceMap `json:"capacity,omitempty" desc:"Replacement capacity override; send an empty object to use runtime-derived capacity."`
	Labels       StringMap   `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations  StringMap   `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// ResourceUnit is an embedded entry of ResourcePool.spec.units[].
type ResourceUnit struct {
	Name         string      `json:"name" desc:"Resource unit (shape) name, unique within the pool."`
	Description  string      `json:"description,omitempty" desc:"Free-text unit description."`
	Requests     ResourceMap `json:"requests" desc:"Resource requests defining the unit shape (e.g. cpu, memory, nvidia.com/gpu)."`
	Limits       ResourceMap `json:"limits" desc:"Resource limits for the unit shape."`
	NodeSelector StringMap   `json:"nodeSelector,omitempty" desc:"Additional node labels narrowing placement for this unit."`
	Annotations  StringMap   `json:"annotations,omitempty" desc:"User-defined annotations."`
}

// ResourceUnitList is a page of ResourceUnit.
type ResourceUnitList struct {
	Items         []ResourceUnit `json:"items" desc:"Resource units in this page."`
	Count         int            `json:"count" binding:"min=0" desc:"Number of units in this page."`
	ContinueToken string         `json:"continueToken,omitempty" desc:"Opaque token to fetch the next page."`
}

// ResourceUnitCreateRequest is the body of POST /resource-pools/{pool}/units.
type ResourceUnitCreateRequest struct {
	Name         string      `json:"name" binding:"required,dns1123,min=1,max=40" desc:"Resource unit (shape) name, unique within the pool."`
	Description  string      `json:"description,omitempty" binding:"max=1000" desc:"Free-text unit description."`
	Requests     ResourceMap `json:"requests" binding:"required" desc:"Resource requests defining the unit shape (e.g. cpu, memory, nvidia.com/gpu)."`
	Limits       ResourceMap `json:"limits" binding:"required" desc:"Resource limits for the unit shape."`
	NodeSelector StringMap   `json:"nodeSelector,omitempty" desc:"Additional node labels narrowing placement for this unit."`
	Annotations  StringMap   `json:"annotations,omitempty" desc:"User-defined annotations."`
}

// ResourceUnitPatchRequest patches a single unit.
type ResourceUnitPatchRequest struct {
	Description  string      `json:"description,omitempty" binding:"max=1000" desc:"Updated free-text unit description."`
	Requests     ResourceMap `json:"requests,omitempty" desc:"Replacement resource requests."`
	Limits       ResourceMap `json:"limits,omitempty" desc:"Replacement resource limits."`
	NodeSelector StringMap   `json:"nodeSelector,omitempty" desc:"Replacement node selector."`
	Annotations  StringMap   `json:"annotations,omitempty" desc:"Replacement annotation set."`
}
