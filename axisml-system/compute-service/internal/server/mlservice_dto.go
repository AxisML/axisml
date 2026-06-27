package server

import (
	"time"

	"github.com/google/uuid"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

// MLServiceCreateRequest is the API request body. Caller selects pool/unit by
// NAME (the ResourcePool CRD lives in K8s; compute reads it via Informer
// cache). `Quota` is the ElasticQuota CR name (cluster-unique string) Compute
// stamps onto Pod labels — opaque, no existence check. `Kind` distinguishes a
// regular online service from a Platform workspace; immutable after create.
// A kind=workspace service's durable volume is pre-provisioned by Platform via
// cluster-manager and referenced as a PVC entry in roles[0].template.volumes;
// compute does not provision storage.
type MLServiceCreateRequest struct {
	Name          string                       `json:"name" binding:"required,axisml_name"`
	Kind          string                       `json:"kind,omitempty"`
	DisplayName   string                       `json:"displayName"`
	Description   string                       `json:"description"`
	Labels        map[string]string            `json:"labels,omitempty"`
	Annotations   map[string]string            `json:"annotations,omitempty"`
	PoolName      string                       `json:"poolName" binding:"required"`
	UnitName      string                       `json:"unitName" binding:"required"`
	Quota         string                       `json:"quota" binding:"required"`
	PriorityClass string                       `json:"priorityClass,omitempty"`
	Backend       *mlservicev1alpha1.Backend   `json:"backend"`
	Roles         []mlservicev1alpha1.RoleSpec `json:"roles" binding:"required,min=1"`
	RunPolicy     *mlservicev1alpha1.RunPolicy `json:"runPolicy"`
	Route         *mlservicev1alpha1.Route     `json:"route"`
}

// MLServiceScaleRequest is the body for /:scale.
type MLServiceScaleRequest struct {
	Replicas int32 `json:"replicas" binding:"required,gte=0"`
}

// MLService is the HTTP response. Mirrors the design yaml: nested spec / status
// sub-trees, phase at the top level, owner / labels / annotations as
// first-class fields. generation / observedGeneration support the K8s-
// style sync signal.
type MLService struct {
	ID                 uuid.UUID                       `json:"id"`
	Namespace          string                          `json:"namespace"`
	Name               string                          `json:"name"`
	Kind               string                          `json:"kind"`
	DisplayName        string                          `json:"displayName,omitempty"`
	Description        string                          `json:"description,omitempty"`
	Owner              string                          `json:"owner,omitempty"`
	Labels             map[string]string               `json:"labels,omitempty"`
	Annotations        map[string]string               `json:"annotations,omitempty"`
	Generation         int64                           `json:"generation"`
	ObservedGeneration int64                           `json:"observedGeneration"`
	Phase              string                          `json:"phase"`
	Spec               mlservicev1alpha1.MLServiceSpec `json:"spec"`
	Status             MLServiceStatus                 `json:"status"`
	CreatedAt          time.Time                       `json:"createdAt"`
	UpdatedAt          time.Time                       `json:"updatedAt"`
	DeletedAt          *time.Time                      `json:"deletedAt,omitempty"`
}

// MLServicePatchRequest is the body for PATCH
// /api/v1/namespaces/{ns}/mlservices/{svc}. Only the four display-tier fields
// are mutable post-create per compute-service.md §4.4; spec mutations go
// through /scale.
type MLServicePatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MLServiceStatus mirrors the CR status sub-tree compute persists for
// MLServices.
type MLServiceStatus struct {
	Message       string               `json:"message,omitempty"`
	ReadyReplicas int32                `json:"readyReplicas"`
	Endpoint      string               `json:"endpoint,omitempty"`
	Conditions    []MLServiceCondition `json:"conditions,omitempty"`
}

// MLServiceCondition is one entry inside an MLService's status.conditions[].
type MLServiceCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
