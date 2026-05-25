package job

import "testing"

func TestIsTerminal_UnknownStatus(t *testing.T) {
	if IsTerminal("unknown") {
		t.Error("unknown status should not be terminal")
	}
	if IsTerminal("") {
		t.Error("empty status should not be terminal")
	}
}

func TestJob_TableName(t *testing.T) {
	if (Job{}).TableName() != "jobs" {
		t.Error("table name mismatch")
	}
}
