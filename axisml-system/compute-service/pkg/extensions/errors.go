package extensions

import (
	"errors"
	"fmt"
)

// ResourceUnavailableError marks an Apply that could not place the workload
// because the runtime currently has insufficient free resources (e.g. no free
// GPU). It is not a failure: the reconciler keeps the workload Pending and
// retries on the next tick. The Kubernetes runtime never returns it (the
// in-cluster scheduler handles pending placement); it originates only in the
// single-host standalone runtime.
type ResourceUnavailableError struct{ Msg string }

func (e *ResourceUnavailableError) Error() string { return e.Msg }

// NewResourceUnavailable builds a ResourceUnavailableError from a format string.
func NewResourceUnavailable(format string, args ...any) error {
	return &ResourceUnavailableError{Msg: fmt.Sprintf(format, args...)}
}

// IsResourceUnavailable reports whether err is, or wraps, a
// ResourceUnavailableError.
func IsResourceUnavailable(err error) bool {
	var e *ResourceUnavailableError
	return errors.As(err, &e)
}
