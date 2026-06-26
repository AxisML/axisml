package logging

import "testing"

func TestNew_ConsoleAndJSON(t *testing.T) {
	con, err := New("debug", "console")
	if err != nil {
		t.Fatalf("console: %v", err)
	}
	if con.GetSink() == nil {
		t.Error("expected non-nil console sink")
	}
	con.Info("hello", "k", "v")

	js, err := New("info", "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if js.GetSink() == nil {
		t.Error("expected non-nil json sink")
	}
}

func TestNew_BadLevelFallsBack(t *testing.T) {
	log, err := New("nonsense", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.GetSink() == nil {
		t.Error("expected non-nil sink despite bad level")
	}
}
