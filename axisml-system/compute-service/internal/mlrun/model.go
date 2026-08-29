package mlrun

// Status enumerates the job state machine (design §4.3).
type Status string

const (
	StatusQueued    Status = "Queued"
	StatusCreating  Status = "Creating"
	StatusPending   Status = "Pending"
	StatusRunning   Status = "Running"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusCanceling Status = "Canceling"
	StatusCancelled Status = "Cancelled"
	StatusDeleting  Status = "Deleting"
	StatusDeleted   Status = "Deleted"
)

func IsTerminal(s Status) bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusCancelled, StatusDeleted:
		return true
	}
	return false
}
