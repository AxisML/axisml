package server

import "time"

// Problem is an RFC 7807 problem-details body. All fields are optional.
type Problem struct {
	Type     URI                 `json:"type,omitempty"`
	Title    string              `json:"title,omitempty"`
	Status   int                 `json:"status,omitempty" binding:"min=100,max=599"`
	Detail   string              `json:"detail,omitempty"`
	Instance string              `json:"instance,omitempty"`
	Code     string              `json:"code,omitempty"`
	Errors   []ProblemFieldError `json:"errors,omitempty"`
}

// ProblemFieldError is one field-level validation issue.
type ProblemFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// HealthStatus is the liveness/readiness probe body.
type HealthStatus struct {
	Status     HealthState       `json:"status"`
	Components map[string]string `json:"components,omitempty"`
}

// EnvVar is a container environment variable (optionally sourced).
type EnvVar struct {
	Name      string         `json:"name"`
	Value     string         `json:"value,omitempty"`
	ValueFrom map[string]any `json:"valueFrom,omitempty"`
}

// Condition is a Kubernetes-style status condition.
type Condition struct {
	Type               string          `json:"type"`
	Status             ConditionStatus `json:"status"`
	Reason             string          `json:"reason,omitempty"`
	Message            string          `json:"message,omitempty"`
	LastTransitionTime time.Time       `json:"lastTransitionTime,omitempty"`
}
