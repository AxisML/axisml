package mlservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
)

type fakePolicyReferenceChecker struct {
	name string
	err  error
}

func TestPhaseViewProjectsAdmittedReplicasWithoutLoadingSpec(t *testing.T) {
	view := phaseView(&store.MLService{
		Name: "predictor", Phase: string(StatusDegraded),
		AdmittedReplicas: datatypes.JSON(`[2]`), StatusJSON: datatypes.JSON(`{"readyReplicas":1}`),
	})
	assert.Equal(t, int32(2), view.AdmittedReplicas)
	assert.Equal(t, int32(1), view.ReadyReplicas)
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
