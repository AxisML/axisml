package mlservice

import "testing"

func TestMLService_TableName(t *testing.T) {
	if (MLService{}).TableName() != "mlservices" {
		t.Error("table name mismatch")
	}
}

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
