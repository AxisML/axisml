package handler

import (
	"context"
	"testing"
	"time"
)

type stubHandler struct{ kind string }

func (s *stubHandler) Kind() string                                 { return s.kind }
func (s *stubHandler) StorageKind() StorageKind                     { return StorageOCI }
func (s *stubHandler) BuildStorageURI(_, _, _ string) string        { return "" }
func (s *stubHandler) BuildPullURI(_, _, _, _ string) string        { return "" }
func (s *stubHandler) ValidateSpec(_ context.Context, _ Spec) error { return nil }
func (s *stubHandler) InitiateUpload(_ context.Context, _ Artifact, _ time.Duration) (Credentials, error) {
	return Credentials{}, nil
}
func (s *stubHandler) IssuePullCredentials(_ context.Context, _ Artifact, _ time.Duration) (Credentials, error) {
	return Credentials{}, nil
}
func (s *stubHandler) VerifyComplete(_ context.Context, _ Artifact, _ CompleteClaim) (string, error) {
	return "", nil
}
func (s *stubHandler) GCBackend(_ context.Context, _ Artifact) error { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Cleanup(reset)
	reset()

	Register(&stubHandler{kind: "model"})
	if _, ok := Get("model"); !ok {
		t.Fatalf("expected model to be registered")
	}
	if _, ok := Get("dataset"); ok {
		t.Fatalf("did not expect dataset to be registered")
	}
}

func TestRegistry_DuplicateRegistrationPanics(t *testing.T) {
	t.Cleanup(reset)
	reset()

	Register(&stubHandler{kind: "model"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate registration")
		}
	}()
	Register(&stubHandler{kind: "model"})
}
