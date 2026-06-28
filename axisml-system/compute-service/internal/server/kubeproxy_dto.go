package server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pod is the stable projection of a Kubernetes Pod returned to clients.
type Pod struct {
	Name      string            `json:"name" desc:"Pod name."`
	Namespace string            `json:"namespace" desc:"Namespace the pod runs in."`
	Phase     string            `json:"phase" desc:"Pod lifecycle phase (Pending, Running, Succeeded, Failed, Unknown)."`
	NodeName  string            `json:"nodeName,omitempty" desc:"Node the pod is scheduled onto."`
	Labels    map[string]string `json:"labels,omitempty" desc:"Pod labels."`
}

// Event projects an events.k8s.io/v1 Event down to its useful fields.
type Event struct {
	Reason    string       `json:"reason" desc:"Short machine-readable reason for the event (e.g. Scheduled, Pulled)."`
	Note      string       `json:"note,omitempty" desc:"Human-readable description of the event."`
	Type      string       `json:"type" desc:"Event type (Normal or Warning)."`
	Object    string       `json:"object" desc:"Target object as \"<kind>/<name>\"."` // "<kind>/<name>"
	Reporter  string       `json:"reportingController,omitempty" desc:"Controller that reported the event."`
	EventTime *metav1.Time `json:"eventTime,omitempty" desc:"Time the event was first observed."`
}
