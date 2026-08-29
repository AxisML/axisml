package rundef

import (
	"strconv"

	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

// ValidatePriorityAnnotations validates the one reserved Run annotation while
// leaving all other user annotations opaque.
func ValidatePriorityAnnotations(annotations map[string]string) error {
	raw, ok := annotations[server.MLRunPriorityAnnotation]
	if !ok {
		return nil
	}
	if _, err := strconv.ParseInt(raw, 10, 32); err != nil {
		return apperrors.New(apperrors.ClassValidation,
			"annotation scheduling.axisml.io/priority must be a base-10 signed int32").
			WithReason("run-priority-invalid")
	}
	return nil
}
