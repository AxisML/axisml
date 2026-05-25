package logging

import "testing"

func TestNew_DevAndProd(t *testing.T) {
	dev, err := New(true)
	if err != nil {
		t.Fatalf("dev: %v", err)
	}
	if dev.GetSink() == nil {
		t.Error("expected non-nil dev sink")
	}
	dev.Info("hello", "k", "v")

	prod, err := New(false)
	if err != nil {
		t.Fatalf("prod: %v", err)
	}
	if prod.GetSink() == nil {
		t.Error("expected non-nil prod sink")
	}
}
