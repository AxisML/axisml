package server

import (
	"time"

	"github.com/google/uuid"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlservice/v1alpha1"
)

// MLServiceCreateRequest is the API request body. Caller selects pool/unit by
// NAME (the ResourcePool CRD lives in K8s; compute reads it via Informer
// cache). Compute derives the tenant's single ElasticQuota for that pool.
// `Kind` distinguishes a
// regular online service from a Platform workspace; immutable after create.
// A kind=workspace service's durable volume is pre-provisioned by Platform via
// cluster-manager and referenced as a PVC entry in roles[0].template.volumes;
// compute does not provision storage.
type MLServiceCreateRequest struct {
	Name          string                       `json:"name" binding:"required,axisml_name" desc:"MLService name, unique within the namespace."`
	Kind          string                       `json:"kind,omitempty" desc:"Service kind (service, workspace, tensorboard); immutable after create, defaults to service."`
	DisplayName   string                       `json:"displayName" desc:"Human-readable service label."`
	Description   string                       `json:"description" desc:"Free-text service description."`
	Labels        map[string]string            `json:"labels,omitempty" desc:"User-defined labels stored on the row and stamped onto the CR."`
	Annotations   map[string]string            `json:"annotations,omitempty" desc:"User-defined annotations stored on the row and stamped onto the CR."`
	PoolName      string                       `json:"poolName" binding:"required" desc:"Resource pool name resolved against the ResourcePool CRD via the Informer cache."`
	UnitName      string                       `json:"unitName" binding:"required" desc:"Resource unit (shape) name within the selected pool."`
	PriorityClass string                       `json:"priorityClass,omitempty" desc:"Optional Kubernetes PriorityClass name for the service's pods."`
	Backend       *mlservicev1alpha1.Backend   `json:"backend" desc:"Compute backend/engine that serves the workload; defaults to (native, deployment) when omitted."`
	Roles         []mlservicev1alpha1.RoleSpec `json:"roles" binding:"required,min=1" desc:"Service topology roles (at least one)."`
	RunPolicy     *mlservicev1alpha1.RunPolicy `json:"runPolicy" desc:"Service-level lifecycle controls (progress deadline)."`
	Route         *mlservicev1alpha1.Route     `json:"route" desc:"Optional external entrypoint (HTTPRoute plus auth/rate-limit policies)."`
}

// MLServiceScaleRequest is the body for /:scale.
type MLServiceScaleRequest struct {
	Replicas int32 `json:"replicas" binding:"required,gte=0" desc:"Desired replica count for the service's primary role."`
}

// MLService is the HTTP response. Mirrors the design yaml: nested spec / status
// sub-trees, phase at the top level, owner / labels / annotations as
// first-class fields. generation / observedGeneration support the K8s-
// style sync signal.
type MLService struct {
	ID                 uuid.UUID                       `json:"id" desc:"Stable service identifier (PG row UUID)."`
	Namespace          string                          `json:"namespace" desc:"Namespace (= tenant identifier) the service belongs to."`
	Name               string                          `json:"name" desc:"MLService name, unique within the namespace."`
	Kind               string                          `json:"kind" desc:"Service kind (service, workspace, tensorboard)."`
	DisplayName        string                          `json:"displayName,omitempty" desc:"Human-readable service label."`
	Description        string                          `json:"description,omitempty" desc:"Free-text service description."`
	Owner              string                          `json:"owner,omitempty" desc:"Username of the service owner."`
	Labels             map[string]string               `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations        map[string]string               `json:"annotations,omitempty" desc:"User-defined annotations."`
	Generation         int64                           `json:"generation" desc:"Desired-state generation, bumped on every spec-affecting change (scale)."`
	ObservedGeneration int64                           `json:"observedGeneration" desc:"Generation the operator last reconciled; equals generation when in sync."`
	Phase              string                          `json:"phase" desc:"Current service lifecycle phase: Creating, Pending, Ready, Degraded, Failed, Deleting, Deleted."`
	Spec               mlservicev1alpha1.MLServiceSpec `json:"spec" desc:"Resolved MLService spec sub-tree (backend, scheduling, roles, route)."`
	Status             MLServiceStatus                 `json:"status" desc:"Operator-reported status sub-tree."`
	CreatedAt          time.Time                       `json:"createdAt" desc:"Time the service was created."`
	UpdatedAt          time.Time                       `json:"updatedAt" desc:"Time the service was last updated."`
	DeletedAt          *time.Time                      `json:"deletedAt,omitempty" desc:"Soft-deletion timestamp, set once the service is deleted."`
}

// MLServicePatchRequest is the body for PATCH
// /api/v1/namespaces/{ns}/mlservices/{svc}. Only the four display-tier fields
// are mutable post-create per compute-service.md §4.4; spec mutations go
// through /scale.
type MLServicePatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty" desc:"Updated human-readable service label."`
	Description *string           `json:"description,omitempty" desc:"Updated free-text service description."`
	Labels      map[string]string `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// MLServiceStatus mirrors the CR status sub-tree compute persists for
// MLServices.
type MLServiceStatus struct {
	Message       string `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	ReadyReplicas int32  `json:"readyReplicas" desc:"Number of replicas that have passed readiness."`
	Endpoint      string `json:"endpoint,omitempty" desc:"Resolved external endpoint URL when a route is enabled."`
}

// MLServicePhase is the lightweight response for the phase probes — GET
// /api/v1/namespaces/{ns}/mlservices/{svc}/phase (single) and the batch GET
// /api/v1/namespaces/{ns}/mlservices/phases. It returns only the service's
// lifecycle phase, readiness and sync signal, skipping the heavy spec sub-tree
// the full MLService payload carries. The generation / observedGeneration pair
// lets a caller tell when a scale has been reconciled (observedGeneration ==
// generation). `name` identifies the service in batch responses.
type MLServicePhase struct {
	Name               string `json:"name" desc:"MLService name, unique within the namespace."`
	Phase              string `json:"phase" desc:"Current service lifecycle phase: Creating, Pending, Ready, Degraded, Failed, Deleting, Deleted."`
	Message            string `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	ReadyReplicas      int32  `json:"readyReplicas" desc:"Number of replicas that have passed readiness."`
	Generation         int64  `json:"generation" desc:"Desired-state generation, bumped on every spec-affecting change (scale)."`
	ObservedGeneration int64  `json:"observedGeneration" desc:"Generation the operator last reconciled; equals generation when in sync."`
}
