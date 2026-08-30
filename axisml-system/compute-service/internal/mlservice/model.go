package mlservice

// Status enumerates the service state machine.
type Status string

const (
	StatusQueued   Status = "Queued"
	StatusCreating Status = "Creating"
	StatusPending  Status = "Pending"
	StatusReady    Status = "Ready"
	StatusDegraded Status = "Degraded"
	StatusFailed   Status = "Failed"
	StatusDeleting Status = "Deleting"
	StatusDeleted  Status = "Deleted"
)
