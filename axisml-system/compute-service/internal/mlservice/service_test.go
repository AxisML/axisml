package mlservice

import (
	"context"
	"testing"

	apperrors "github.com/axisml/axisml/components/compute-service/pkg/errors"
)

type fakePolicyReferenceChecker struct {
	name string
	err  error
}

func (f fakePolicyReferenceChecker) ActiveReferenceName(context.Context, string, string) (string, error) {
	return f.name, f.err
}

func TestDeleteRejectsServiceReferencedByTrafficPolicy(t *testing.T) {
	err := checkActivePolicyReference(
		context.Background(),
		fakePolicyReferenceChecker{name: "canary"},
		"tenant-a",
		"inference",
	)
	apiErr, ok := apperrors.As(err)
	if !ok || apiErr.Code != apperrors.CodeConflict {
		t.Fatalf("checkActivePolicyReference() error = %v, want conflict", err)
	}
}

func TestDeleteAllowsUnreferencedService(t *testing.T) {
	err := checkActivePolicyReference(
		context.Background(),
		fakePolicyReferenceChecker{},
		"tenant-a",
		"inference",
	)
	if err != nil {
		t.Fatalf("checkActivePolicyReference() error = %v, want nil", err)
	}
}
