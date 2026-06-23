package trafficpolicy

// Status enumerates the traffic-policy state machine
// (compute-service.md §4.5).
type Status string

const (
	StatusCreating Status = "Creating"
	StatusPending  Status = "Pending"
	StatusReady    Status = "Ready"
	StatusDegraded Status = "Degraded"
	StatusFailed   Status = "Failed"
	StatusDeleting Status = "Deleting"
	StatusDeleted  Status = "Deleted"
)
