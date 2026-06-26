package logging

import "testing"

func TestNew_Console(t *testing.T) {
	log, err := New("debug", "console")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.GetSink() == nil {
		t.Error("expected non-nil sink for console logger")
	}
	log.Info("hello", "k", "v")
}

func TestNew_JSON(t *testing.T) {
	log, err := New("info", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.GetSink() == nil {
		t.Error("expected non-nil sink for json logger")
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
