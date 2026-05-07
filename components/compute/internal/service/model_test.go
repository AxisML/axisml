package service

import "testing"

func TestService_TableName(t *testing.T) {
	if (Service{}).TableName() != "services" {
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
