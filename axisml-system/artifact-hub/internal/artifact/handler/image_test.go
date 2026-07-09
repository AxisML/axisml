package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

func TestImageHandler_Basics(t *testing.T) {
	h := NewImageHandler(oci.New(oci.Config{Endpoint: "zot.local:5000"}))
	assert.Equal(t, "image", h.Kind())
	assert.Equal(t, StorageOCI, h.StorageKind())
	// Images use the "images" sub-path (distinct from models).
	assert.Equal(t, "zot.local:5000/namespaces/ns/images/n:v", h.BuildStorageURI("ns", "n", "v"))
}

func TestImageHandler_ValidateSpec(t *testing.T) {
	h := NewImageHandler(oci.New(oci.Config{Endpoint: "zot.local:5000"}))
	cases := []struct {
		name    string
		spec    Spec
		wantErr bool
	}{
		{name: "training ok", spec: Spec{"purpose": "training"}},
		{name: "inference ok", spec: Spec{"purpose": "inference"}},
		{name: "dev ok", spec: Spec{"purpose": "dev"}},
		{name: "missing purpose", spec: Spec{}, wantErr: true},
		{name: "empty purpose", spec: Spec{"purpose": ""}, wantErr: true},
		{name: "non-string purpose", spec: Spec{"purpose": 7}, wantErr: true},
		{name: "unsupported purpose", spec: Spec{"purpose": "prod"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := h.ValidateSpec(context.Background(), tc.spec)
			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			e, ok := apperrors.As(err)
			require.True(t, ok)
			assert.Equal(t, apperrors.CodeValidation, e.Code)
		})
	}
}
