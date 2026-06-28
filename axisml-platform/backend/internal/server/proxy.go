package server

import "time"

// Pod is a Platform projection of a backend pod (read-only).
type Pod struct {
	Name         string     `json:"name" desc:"Pod name."`
	Role         string     `json:"role,omitempty" desc:"Topology role the pod belongs to (e.g. master, worker)."`
	ReplicaIndex int        `json:"replicaIndex,omitempty" binding:"min=0" desc:"Zero-based replica index within the role."`
	Phase        PodPhase   `json:"phase" desc:"Pod lifecycle phase (Pending, Running, Succeeded, Failed, Unknown)."`
	NodeName     string     `json:"nodeName,omitempty" desc:"Node the pod is scheduled on."`
	StartedAt    *time.Time `json:"startedAt,omitempty" desc:"Time the pod started running."`
	FinishedAt   *time.Time `json:"finishedAt,omitempty" desc:"Time the pod terminated."`
	ExitCode     *int       `json:"exitCode,omitempty" desc:"Exit code of the container's main process."`
	Message      string     `json:"message,omitempty" desc:"Human-readable status detail for the pod."`
	RestartCount int        `json:"restartCount,omitempty" binding:"min=0" desc:"Number of times the pod's container has restarted."`
}

// PodList is a list of Pod.
type PodList struct {
	Items []Pod `json:"items" desc:"Pods in the list."`
	Count int   `json:"count" binding:"min=0" desc:"Number of pods in the list."`
}

// Event is a Platform projection of a Kubernetes event.
type Event struct {
	Type           EventType `json:"type" desc:"Event type (Normal, Warning)."`
	Reason         string    `json:"reason" desc:"Short machine-readable reason for the event."`
	Message        string    `json:"message" desc:"Human-readable event message."`
	Source         string    `json:"source,omitempty" desc:"Component that reported the event."`
	FirstTimestamp time.Time `json:"firstTimestamp,omitempty" desc:"Time the event was first observed."`
	LastTimestamp  time.Time `json:"lastTimestamp" desc:"Time the event was last observed."`
	Count          int       `json:"count,omitempty" binding:"min=0" desc:"Number of times this event has occurred."`
	InvolvedObject struct {
		Kind      string `json:"kind,omitempty" desc:"Kind of the object the event refers to."`
		Name      string `json:"name,omitempty" desc:"Name of the object the event refers to."`
		Namespace string `json:"namespace,omitempty" desc:"Namespace of the object the event refers to."`
	} `json:"involvedObject,omitempty" desc:"Object the event is about."`
}

// EventList is a list of Event.
type EventList struct {
	Items []Event `json:"items" desc:"Events in the list."`
	Count int     `json:"count" binding:"min=0" desc:"Number of events in the list."`
}
