package mlrun

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	mlrunv1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlrun/v1alpha1"
)

func TestToCR_StampsLabelsAndCopiesSpec(t *testing.T) {
	id := uuid.New()
	spec := mlrunv1alpha1.MLRunSpec{
		Backend: mlrunv1alpha1.BackendSpec{Name: "native", Engine: "job"},
		Scheduling: mlrunv1alpha1.SchedulingSpec{
			Quota: "axisml-default",
		},
		Roles: []mlrunv1alpha1.RoleSpec{{
			Name:     mlrunv1alpha1.DefaultRoleName,
			Replicas: 2,
		}},
	}
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)

	row := &MLRun{
		ID:        id,
		Namespace: "team-a",
		Name:      "trainer",
		Spec:      datatypes.JSON(specJSON),
	}

	cr, err := ToCR(row)
	require.NoError(t, err)

	assert.Equal(t, "trainer", cr.Name)
	assert.Equal(t, "team-a", cr.Namespace)
	assert.Equal(t, id.String(), cr.Labels[mlrunv1alpha1.LabelRunID])
	assert.Equal(t, "axisml-default", cr.Labels[mlrunv1alpha1.LabelQuota],
		"quota label is the operator's gating signal — it must be sourced from spec.scheduling.quota")
	assert.Equal(t, spec.Backend, cr.Spec.Backend)
	assert.Equal(t, int32(2), cr.Spec.Roles[0].Replicas)
}

func TestToCR_EmptySpec_StillRenders(t *testing.T) {
	row := &MLRun{
		ID:        uuid.New(),
		Namespace: "team-b",
		Name:      "no-spec",
	}
	cr, err := ToCR(row)
	require.NoError(t, err)
	assert.Equal(t, "no-spec", cr.Name)
	assert.Empty(t, cr.Labels[mlrunv1alpha1.LabelQuota],
		"empty spec leaves quota label empty — operator's Validate will reject downstream")
}

func TestToCR_BadJSON_ReturnsError(t *testing.T) {
	row := &MLRun{
		ID:        uuid.New(),
		Namespace: "team-c",
		Name:      "bad",
		Spec:      datatypes.JSON([]byte(`{"backend":`)), // truncated
	}
	_, err := ToCR(row)
	assert.Error(t, err)
}

func TestIsTerminal(t *testing.T) {
	terminal := []Status{StatusSucceeded, StatusFailed, StatusCancelled, StatusDeleted}
	for _, s := range terminal {
		assert.Truef(t, IsTerminal(s), "%s must be terminal", s)
	}
	nonTerminal := []Status{StatusCreating, StatusPending, StatusRunning, StatusCanceling, StatusDeleting}
	for _, s := range nonTerminal {
		assert.Falsef(t, IsTerminal(s), "%s must not be terminal", s)
	}
}
