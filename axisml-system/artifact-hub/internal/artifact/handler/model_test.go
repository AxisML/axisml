package handler

import (
	"context"
	"testing"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

func TestModelHandler_ValidateSpec(t *testing.T) {
	h := NewModelHandler(oci.New(oci.Config{Endpoint: "zot.local:5000"}))
	cases := []struct {
		name    string
		spec    Spec
		wantErr apperrors.Code
	}{
		{
			name: "happy path",
			spec: Spec{"framework": "pytorch", "format": "application/vnd.pytorch.bin"},
		},
		{
			name:    "missing framework",
			spec:    Spec{"format": "application/x-bin"},
			wantErr: apperrors.CodeValidation,
		},
		{
			name:    "unsupported framework",
			spec:    Spec{"framework": "jax", "format": "application/x-bin"},
			wantErr: apperrors.CodeValidation,
		},
		{
			name:    "missing format",
			spec:    Spec{"framework": "pytorch"},
			wantErr: apperrors.CodeValidation,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := h.ValidateSpec(context.Background(), tc.spec)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %s, got nil", tc.wantErr)
			}
			e, ok := apperrors.As(err)
			if !ok {
				t.Fatalf("expected apperrors.E, got %T", err)
			}
			if e.Code != tc.wantErr {
				t.Fatalf("expected code %s, got %s", tc.wantErr, e.Code)
			}
		})
	}
}

func TestModelHandler_BuildStorageURI(t *testing.T) {
	h := NewModelHandler(oci.New(oci.Config{Endpoint: "zot.local:5000"}))
	got := h.BuildStorageURI("team-a", "llama-7b", "v1")
	want := "zot.local:5000/namespaces/team-a/models/llama-7b:v1"
	if got != want {
		t.Fatalf("BuildStorageURI = %q, want %q", got, want)
	}
}
