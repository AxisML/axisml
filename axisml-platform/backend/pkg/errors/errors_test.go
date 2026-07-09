package errors_test

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

func TestNew(t *testing.T) {
	e := apperrors.New(apperrors.ClassNotFound, "missing")
	assert.Equal(t, apperrors.ClassNotFound, e.Class)
	assert.Equal(t, "missing", e.Message)
	assert.Empty(t, e.Reason)
	// Code falls back to the Class when no Reason is set.
	assert.Equal(t, "not_found", e.Code())
}

func TestNewf(t *testing.T) {
	e := apperrors.Newf(apperrors.ClassValidation, "bad %s=%d", "x", 3)
	assert.Equal(t, apperrors.ClassValidation, e.Class)
	assert.Equal(t, "bad x=3", e.Message)
}

func TestWrap(t *testing.T) {
	cause := stderrors.New("boom")
	e := apperrors.Wrap(apperrors.ClassUpstream, "call failed", cause)
	assert.Equal(t, apperrors.ClassUpstream, e.Class)
	assert.Equal(t, "call failed", e.Message)
	assert.Equal(t, cause, e.Unwrap())
}

func TestError(t *testing.T) {
	tests := []struct {
		name string
		err  *apperrors.E
		want string
	}{
		{
			name: "no cause uses class code",
			err:  apperrors.New(apperrors.ClassConflict, "dup"),
			want: "conflict: dup",
		},
		{
			name: "no cause uses reason code",
			err:  apperrors.New(apperrors.ClassConflict, "dup").WithReason("already-exists"),
			want: "already-exists: dup",
		},
		{
			name: "with cause",
			err:  apperrors.Wrap(apperrors.ClassUpstream, "call failed", stderrors.New("boom")),
			want: "upstream_failure: call failed: boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestCode(t *testing.T) {
	// Reason wins over Class.
	withReason := apperrors.New(apperrors.ClassForbidden, "no").WithReason("tenant-suspended")
	assert.Equal(t, "tenant-suspended", withReason.Code())
	// Class fallback when Reason is empty.
	noReason := apperrors.New(apperrors.ClassForbidden, "no")
	assert.Equal(t, "forbidden", noReason.Code())
}

func TestWithReasonAndDetails(t *testing.T) {
	details := map[string]any{"service": "compute", "status": 502}
	e := apperrors.New(apperrors.ClassUpstream, "x").
		WithReason("upstream-failure").
		WithDetails(details)
	assert.Equal(t, "upstream-failure", e.Reason)
	assert.Equal(t, details, e.Details)
	// Fluent builders return the same instance.
	assert.Same(t, e, e.WithReason("upstream-failure"))
}

func TestUnwrap_NoCause(t *testing.T) {
	e := apperrors.New(apperrors.ClassInternal, "x")
	assert.Nil(t, e.Unwrap())
}

func TestAs_Matching(t *testing.T) {
	e := apperrors.New(apperrors.ClassNotFound, "missing").WithReason("gone-away")
	// Directly.
	got, ok := apperrors.As(e)
	require.True(t, ok)
	assert.Equal(t, "gone-away", got.Code())
	// Through a wrapping chain.
	wrapped := fmt.Errorf("context: %w", e)
	got, ok = apperrors.As(wrapped)
	require.True(t, ok)
	assert.Equal(t, apperrors.ClassNotFound, got.Class)
}

func TestAs_NonMatching(t *testing.T) {
	got, ok := apperrors.As(stderrors.New("plain"))
	assert.False(t, ok)
	assert.Nil(t, got)
}
