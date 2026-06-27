package server

import "time"

// Problem is an RFC 7807 problem-details body. All fields are optional.
type Problem struct {
	Type     URI                 `json:"type,omitempty" desc:"URI reference identifying the problem type."`
	Title    string              `json:"title,omitempty" desc:"Short, human-readable summary of the problem type."`
	Status   int                 `json:"status,omitempty" binding:"min=100,max=599" desc:"HTTP status code for this occurrence of the problem."`
	Detail   string              `json:"detail,omitempty" desc:"Human-readable explanation specific to this occurrence."`
	Instance string              `json:"instance,omitempty" desc:"URI reference identifying the specific occurrence of the problem."`
	Code     string              `json:"code,omitempty" desc:"Stable machine-readable error code."`
	Errors   []ProblemFieldError `json:"errors,omitempty" desc:"Field-level validation errors, when applicable."`
}

// ProblemFieldError is one field-level validation issue.
type ProblemFieldError struct {
	Field   string `json:"field" desc:"Name of the field that failed validation."`
	Message string `json:"message" desc:"Human-readable description of the validation failure."`
}

// HealthStatus is the liveness/readiness probe body.
type HealthStatus struct {
	Status     HealthState       `json:"status" desc:"Overall health state (e.g. ok, degraded)."`
	Components map[string]string `json:"components,omitempty" desc:"Per-dependency health states keyed by component name."`
}

// EnvVar is a container environment variable (optionally sourced).
type EnvVar struct {
	Name      string         `json:"name" desc:"Environment variable name."`
	Value     string         `json:"value,omitempty" desc:"Literal value for the variable."`
	ValueFrom map[string]any `json:"valueFrom,omitempty" desc:"Source the value from a secret/configmap/field (pass-through to the K8s EnvVarSource shape)."`
}

// Condition is a Kubernetes-style status condition.
type Condition struct {
	Type               string          `json:"type" desc:"Condition type (e.g. Ready, Available)."`
	Status             ConditionStatus `json:"status" desc:"Status of the condition (True, False, Unknown)."`
	Reason             string          `json:"reason,omitempty" desc:"Machine-readable reason for the condition's last transition."`
	Message            string          `json:"message,omitempty" desc:"Human-readable detail about the last transition."`
	LastTransitionTime time.Time       `json:"lastTransitionTime,omitempty" desc:"Time the condition last transitioned to this status."`
}
