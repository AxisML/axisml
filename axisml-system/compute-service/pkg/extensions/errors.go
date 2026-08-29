package extensions

import (
	"errors"
	"fmt"
)

// ResourceUnavailableError marks an Apply that could not place the workload
// because the runtime currently has insufficient free resources (e.g. no free
// GPU). For MLRuns it is not a failure: when Apply created no instance, the
// reconciler returns the Run to Queued for a fresh admission decision. The
// Kubernetes runtime never returns it (the
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

// TerminalApplyError marks an ApplyMLRun error that should transition the Run
// to Failed instead of leaving it Pending for another identical retry. Runtime
// implementations should use it only when retrying the unchanged desired Run
// is not useful, for example when its image cannot be pulled.
type TerminalApplyError struct{ Err error }

func (e *TerminalApplyError) Error() string { return e.Err.Error() }

func (e *TerminalApplyError) Unwrap() error { return e.Err }

// NewTerminalApplyError wraps err as a terminal Apply failure. A nil error
// remains nil, and an already-marked error is returned unchanged.
func NewTerminalApplyError(err error) error {
	if err == nil || IsTerminalApplyError(err) {
		return err
	}
	return &TerminalApplyError{Err: err}
}

// IsTerminalApplyError reports whether err, or an error it wraps, marks a
// terminal Apply failure.
func IsTerminalApplyError(err error) bool {
	var e *TerminalApplyError
	return errors.As(err, &e)
}
