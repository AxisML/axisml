package mlservice

import "testing"

func TestStatus_Constants(t *testing.T) {
	for _, s := range []Status{
		StatusCreating, StatusPending, StatusReady,
		StatusDegraded, StatusFailed, StatusDeleting, StatusDeleted,
	} {
		if string(s) == "" {
			t.Error("empty status constant")
		}
	}
}
