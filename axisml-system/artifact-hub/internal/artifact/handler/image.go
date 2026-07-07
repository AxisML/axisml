package handler

import (
	"context"

	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/storage/oci"
	apperrors "github.com/axisml/axisml/axisml-system/artifact-hub/pkg/errors"
)

// validImagePurposes is the closed set per design §4.3.
var validImagePurposes = map[string]struct{}{
	"training":  {},
	"inference": {},
	"dev":       {},
}

// ImageHandler implements Handler for OCI-backed container image artifacts. It
// mirrors ModelHandler against the same zot endpoint via the shared ociBacked
// mechanics but with a distinct repository sub-path, adding only image-specific
// spec validation.
type ImageHandler struct{ ociBacked }

// NewImageHandler returns a Handler implementation.
func NewImageHandler(client *oci.Client) *ImageHandler {
	return &ImageHandler{ociBacked{oci: client, kind: "image", subpath: "images"}}
}

// ValidateSpec enforces the design §4.3 required field (purpose).
func (h *ImageHandler) ValidateSpec(_ context.Context, spec Spec) error {
	purpose, ok := stringField(spec, "purpose")
	if !ok || purpose == "" {
		return apperrors.New(apperrors.CodeValidation, "spec.purpose is required")
	}
	if _, valid := validImagePurposes[purpose]; !valid {
		return apperrors.Newf(apperrors.CodeValidation,
			"spec.purpose %q is not in {training,inference,dev}", purpose)
	}
	return nil
}
