package server

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pod is the stable projection of a Kubernetes Pod returned to clients.
type Pod struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Phase     string            `json:"phase"`
	NodeName  string            `json:"nodeName,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Event projects an events.k8s.io/v1 Event down to its useful fields.
type Event struct {
	Reason    string       `json:"reason"`
	Note      string       `json:"note,omitempty"`
	Type      string       `json:"type"`
	Object    string       `json:"object"` // "<kind>/<name>"
	Reporter  string       `json:"reportingController,omitempty"`
	EventTime *metav1.Time `json:"eventTime,omitempty"`
}
