package server

import (
	"time"

	"github.com/google/uuid"

	mltp "github.com/axisml/axisml/components/compute-operator/api/mltrafficpolicy/v1alpha1"
)

// TrafficPolicyCreateRequest is the API request body. The backend tuple is
// derived (not supplied) from the member services' family.
type TrafficPolicyCreateRequest struct {
	Name        string               `json:"name" binding:"required,axisml_name"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description"`
	Labels      map[string]string    `json:"labels,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	Mode        string               `json:"mode" binding:"required"`
	Endpoint    mltp.Endpoint        `json:"endpoint"`
	Backends    []mltp.BackendMember `json:"backends" binding:"required,min=1"`
}

// TrafficPolicySplitRequest adjusts per-backend weights. Only listed backends
// change.
type TrafficPolicySplitRequest struct {
	Backends []TrafficPolicyWeightUpdate `json:"backends" binding:"required,min=1"`
}

// TrafficPolicyWeightUpdate is one (serviceName, weight) pair.
type TrafficPolicyWeightUpdate struct {
	ServiceName string `json:"serviceName" binding:"required"`
	Weight      int32  `json:"weight"`
}

// TrafficPolicyPatchRequest mutates display-tier metadata only (no CR touch, no
// generation bump).
type TrafficPolicyPatchRequest struct {
	DisplayName *string           `json:"displayName,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// TrafficPolicy is the HTTP response.
type TrafficPolicy struct {
	ID                 uuid.UUID                `json:"id"`
	Namespace          string                   `json:"namespace"`
	Name               string                   `json:"name"`
	Mode               string                   `json:"mode"`
	DisplayName        string                   `json:"displayName,omitempty"`
	Description        string                   `json:"description,omitempty"`
	Owner              string                   `json:"owner,omitempty"`
	Labels             map[string]string        `json:"labels,omitempty"`
	Annotations        map[string]string        `json:"annotations,omitempty"`
	Generation         int64                    `json:"generation"`
	ObservedGeneration int64                    `json:"observedGeneration"`
	Phase              string                   `json:"phase"`
	Spec               mltp.MLTrafficPolicySpec `json:"spec"`
	Status             TrafficPolicyStatus      `json:"status"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
	DeletedAt          *time.Time               `json:"deletedAt,omitempty"`
}

// TrafficPolicyStatus mirrors the CR status sub-tree compute persists for
// traffic policies.
type TrafficPolicyStatus struct {
	Message    string                       `json:"message,omitempty"`
	Endpoint   string                       `json:"endpoint,omitempty"`
	Backends   []TrafficPolicyBackendStatus `json:"backends,omitempty"`
	Conditions []TrafficPolicyCondition     `json:"conditions,omitempty"`
}

// TrafficPolicyBackendStatus is one entry inside a policy's status.backends[].
type TrafficPolicyBackendStatus struct {
	ServiceName string `json:"serviceName"`
	Weight      int32  `json:"weight"`
	Ready       bool   `json:"ready"`
}

// TrafficPolicyCondition is one entry inside a policy's status.conditions[].
type TrafficPolicyCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime,omitempty"`
}
