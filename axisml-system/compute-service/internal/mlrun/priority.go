package mlrun

import (
	"fmt"
	"strconv"

	mlrunv1alpha1 "github.com/axisml/axisml/axisml-system/apis/mlrun/v1alpha1"
)

// PriorityFromAnnotations parses the public queue-priority annotation. Missing
// means the neutral priority zero; malformed and out-of-range values are
// rejected at the API boundary rather than silently reordered.
func PriorityFromAnnotations(annotations map[string]string) (int32, error) {
	raw, ok := annotations[mlrunv1alpha1.AnnotationPriority]
	if !ok {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("annotation %s must be a base-10 signed int32", mlrunv1alpha1.AnnotationPriority)
	}
	return int32(v), nil
}
