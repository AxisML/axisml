package server

import (
	"time"

	"github.com/google/uuid"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
)

// MLRunCreateRequest is the API request body. Caller selects pool/unit by NAME
// (the ResourcePool CRD lives in K8s; compute reads it via Informer cache).
// `Quota` is the ElasticQuota CR name (cluster-unique string) stamped onto
// Pod labels — compute treats it as opaque.
type MLRunCreateRequest struct {
	Name          string                       `json:"name" binding:"required,axisml_name"`
	DisplayName   string                       `json:"displayName"`
	Description   string                       `json:"description"`
	Labels        map[string]string            `json:"labels,omitempty"`
	Annotations   map[string]string            `json:"annotations,omitempty"`
	PoolName      string                       `json:"poolName" binding:"required"`
	UnitName      string                       `json:"unitName" binding:"required"`
	Quota         string                       `json:"quota" binding:"required"`
	PriorityClass string                       `json:"priorityClass,omitempty"`
	Backend       *mlrunv1alpha1.BackendSpec   `json:"backend"`
	Roles         []mlrunv1alpha1.RoleSpec     `json:"roles" binding:"required,min=1"`
	RunPolicy     *mlrunv1alpha1.RunPolicySpec `json:"runPolicy"`
}

// MLRun is the HTTP response payload. Mirrors the design yaml: nested
// spec / status sub-trees, phase at the top level, owner / labels /
// annotations carried separately.
type MLRun struct {
	ID          uuid.UUID               `json:"id"`
	Namespace   string                  `json:"namespace"`
	Name        string                  `json:"name"`
	DisplayName string                  `json:"displayName,omitempty"`
	Description string                  `json:"description,omitempty"`
	Owner       string                  `json:"owner,omitempty"`
	Labels      map[string]string       `json:"labels,omitempty"`
	Annotations map[string]string       `json:"annotations,omitempty"`
	Phase       string                  `json:"phase"`
	Spec        mlrunv1alpha1.MLRunSpec `json:"spec"`
	Status      MLRunStatus             `json:"status"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
	DeletedAt   *time.Time              `json:"deletedAt,omitempty"`
}

// MLRunPatchRequest is the body for PATCH /api/v1/namespaces/{ns}/mlruns/{mlrun}.
// Per design §4.3, only the four "PG-only display" fields are mutable
// after create — the rest of the spec is frozen.
type MLRunPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// MLRunStatus mirrors the CR status sub-tree compute persists for MLRuns.
type MLRunStatus struct {
	Message    string           `json:"message,omitempty"`
	StartedAt  *time.Time       `json:"startedAt,omitempty"`
	FinishedAt *time.Time       `json:"finishedAt,omitempty"`
	Conditions []MLRunCondition `json:"conditions,omitempty"`
}

// MLRunCondition is one entry inside an MLRun's status.conditions[].
type MLRunCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
