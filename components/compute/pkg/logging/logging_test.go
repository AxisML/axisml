package logging

import "testing"

func TestNew_Development(t *testing.T) {
	log, err := New(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.GetSink() == nil {
		t.Error("expected non-nil sink for development logger")
	}
	log.Info("hello", "k", "v")
}

func TestNew_Production(t *testing.T) {
	log, err := New(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if log.GetSink() == nil {
		t.Error("expected non-nil sink for production logger")
	}
}
