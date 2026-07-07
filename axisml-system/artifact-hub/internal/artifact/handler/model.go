package handler

import (
	"context"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// validFrameworks is the closed set design §5.1 calls out for the model Kind.
var validFrameworks = map[string]struct{}{
	"pytorch":     {},
	"tensorflow":  {},
	"onnx":        {},
	"safetensors": {},
	"gguf":        {},
	"custom":      {},
}

// ModelHandler implements Handler for OCI-backed model artifacts. It adds
// model-specific spec validation on top of the shared ociBacked mechanics.
type ModelHandler struct{ ociBacked }

// NewModelHandler returns a Handler implementation. Caller must pass a
// configured OCI client.
func NewModelHandler(client *oci.Client) *ModelHandler {
	return &ModelHandler{ociBacked{oci: client, kind: "model", subpath: "models"}}
}

// ValidateSpec enforces the design §5.1 required fields.
func (h *ModelHandler) ValidateSpec(_ context.Context, spec Spec) error {
	framework, ok := stringField(spec, "framework")
	if !ok {
		return apperrors.New(apperrors.CodeValidation, "spec.framework is required")
	}
	if _, valid := validFrameworks[framework]; !valid {
		return apperrors.Newf(apperrors.CodeValidation,
			"spec.framework %q is not in {pytorch,tensorflow,onnx,safetensors,gguf,custom}", framework)
	}
	if format, ok := stringField(spec, "format"); !ok || format == "" {
		return apperrors.New(apperrors.CodeValidation, "spec.format is required")
	}
	return nil
}
