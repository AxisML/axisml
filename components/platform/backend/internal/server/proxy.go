package server

import "time"

// Pod is a Platform projection of a backend pod (read-only).
type Pod struct {
	Name         string     `json:"name"`
	Role         string     `json:"role,omitempty"`
	ReplicaIndex int        `json:"replicaIndex,omitempty" binding:"min=0"`
	Phase        PodPhase   `json:"phase"`
	NodeName     string     `json:"nodeName,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	Message      string     `json:"message,omitempty"`
	RestartCount int        `json:"restartCount,omitempty" binding:"min=0"`
}

// PodList is a list of Pod.
type PodList struct {
	Items []Pod `json:"items"`
	Count int   `json:"count" binding:"min=0"`
}

// Event is a Platform projection of a Kubernetes event.
type Event struct {
	Type           EventType `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Source         string    `json:"source,omitempty"`
	FirstTimestamp time.Time `json:"firstTimestamp,omitempty"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
	Count          int       `json:"count,omitempty" binding:"min=0"`
	InvolvedObject struct {
		Kind      string `json:"kind,omitempty"`
		Name      string `json:"name,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"involvedObject,omitempty"`
}

// EventList is a list of Event.
type EventList struct {
	Items []Event `json:"items"`
	Count int     `json:"count" binding:"min=0"`
}
