package server

import (
	"time"

	"github.com/google/uuid"

	mltp "github.com/axisml/axisml/axisml-system/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// TrafficPolicyCreateRequest is the API request body. The backend tuple is
// derived (not supplied) from the member services' family.
type TrafficPolicyCreateRequest struct {
	Name        string               `json:"name" binding:"required,axisml_name" desc:"Traffic policy name, unique within the namespace."`
	DisplayName string               `json:"displayName" desc:"Human-readable policy label."`
	Description string               `json:"description" desc:"Free-text policy description."`
	Labels      map[string]string    `json:"labels,omitempty" desc:"User-defined labels stored on the row and stamped onto the CR."`
	Annotations map[string]string    `json:"annotations,omitempty" desc:"User-defined annotations stored on the row and stamped onto the CR."`
	Mode        string               `json:"mode" binding:"required" desc:"Traffic split mode (weighted, canary, bluegreen); immutable after create."`
	Endpoint    mltp.Endpoint        `json:"endpoint" desc:"Stable external entrypoint (path, hostname, auth) shared by all members."`
	Backends    []mltp.BackendMember `json:"backends" binding:"required,min=1" desc:"Member MLServices and their weights (at least one)."`
}

// TrafficPolicySplitRequest adjusts per-backend weights. Only listed backends
// change.
type TrafficPolicySplitRequest struct {
	Backends []TrafficPolicyWeightUpdate `json:"backends" binding:"required,min=1" desc:"Per-backend weight updates; only listed backends change."`
}

// TrafficPolicyWeightUpdate is one (serviceName, weight) pair.
type TrafficPolicyWeightUpdate struct {
	ServiceName string `json:"serviceName" binding:"required" desc:"Member MLService name whose weight is being set."`
	Weight      int32  `json:"weight" desc:"New weight for the member (weights across members sum to 100)."`
}

// TrafficPolicyPatchRequest mutates display-tier metadata only (no CR touch, no
// generation bump).
type TrafficPolicyPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty" desc:"Updated human-readable policy label."`
	Description *string           `json:"description,omitempty" desc:"Updated free-text policy description."`
	Labels      map[string]string `json:"labels,omitempty" desc:"Replacement label set."`
	Annotations map[string]string `json:"annotations,omitempty" desc:"Replacement annotation set."`
}

// TrafficPolicy is the HTTP response.
type TrafficPolicy struct {
	ID                 uuid.UUID                `json:"id" desc:"Stable policy identifier (PG row UUID)."`
	Namespace          string                   `json:"namespace" desc:"Namespace (= tenant identifier) the policy belongs to."`
	Name               string                   `json:"name" desc:"Traffic policy name, unique within the namespace."`
	Mode               string                   `json:"mode" desc:"Traffic split mode (weighted, canary, bluegreen)."`
	DisplayName        string                   `json:"displayName,omitempty" desc:"Human-readable policy label."`
	Description        string                   `json:"description,omitempty" desc:"Free-text policy description."`
	Owner              string                   `json:"owner,omitempty" desc:"Username of the policy owner."`
	Labels             map[string]string        `json:"labels,omitempty" desc:"User-defined labels."`
	Annotations        map[string]string        `json:"annotations,omitempty" desc:"User-defined annotations."`
	Generation         int64                    `json:"generation" desc:"Desired-state generation, bumped on every spec-affecting change (split, promote, rollback)."`
	ObservedGeneration int64                    `json:"observedGeneration" desc:"Generation the operator last reconciled; equals generation when in sync."`
	Phase              string                   `json:"phase" desc:"Current policy lifecycle phase (Pending, Ready, Degraded, Failed)."`
	Spec               mltp.MLTrafficPolicySpec `json:"spec" desc:"Resolved MLTrafficPolicy spec sub-tree (backend, mode, endpoint, members)."`
	Status             TrafficPolicyStatus      `json:"status" desc:"Operator-reported status sub-tree."`
	CreatedAt          time.Time                `json:"createdAt" desc:"Time the policy was created."`
	UpdatedAt          time.Time                `json:"updatedAt" desc:"Time the policy was last updated."`
	DeletedAt          *time.Time               `json:"deletedAt,omitempty" desc:"Soft-deletion timestamp, set once the policy is deleted."`
}

// TrafficPolicyStatus mirrors the CR status sub-tree compute persists for
// traffic policies.
type TrafficPolicyStatus struct {
	Message  string                       `json:"message,omitempty" desc:"Human-readable status detail for the current phase."`
	Endpoint string                       `json:"endpoint,omitempty" desc:"Resolved external endpoint URL fronting the member services."`
	Backends []TrafficPolicyBackendStatus `json:"backends,omitempty" desc:"Per-member effective weight and readiness."`
}

// TrafficPolicyBackendStatus is one entry inside a policy's status.backends[].
type TrafficPolicyBackendStatus struct {
	ServiceName string `json:"serviceName" desc:"Member MLService name."`
	Weight      int32  `json:"weight" desc:"Effective traffic weight currently routed to the member."`
	Ready       bool   `json:"ready" desc:"True when the member service is ready to receive traffic."`
}
