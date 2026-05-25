package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	mlservicev1alpha1 "github.com/axisml/axisml/components/compute-operator/api/mlservice/v1alpha1"
)

func TestServiceToCR_StampsLabelsAndCopiesSpec(t *testing.T) {
	id := uuid.New()
	spec := mlservicev1alpha1.MLServiceSpec{
		Backend:    mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"},
		Scheduling: mlservicev1alpha1.Scheduling{Quota: "axisml-default"},
		ModelRef:   mlservicev1alpha1.ModelRef{Name: "demo", Version: "v1"},
		Roles: []mlservicev1alpha1.RoleSpec{{
			Name:     mlservicev1alpha1.DefaultRoleName,
			Replicas: 3,
		}},
	}
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)

	row := &Service{
		ID:        id,
		Namespace: "team-a",
		Name:      "predictor",
		Spec:      datatypes.JSON(specJSON),
	}

	cr, err := ToCR(row)
	require.NoError(t, err)

	assert.Equal(t, "predictor", cr.Name)
	assert.Equal(t, "team-a", cr.Namespace)
	assert.Equal(t, id.String(), cr.Labels[mlservicev1alpha1.LabelServiceID])
	assert.Equal(t, "axisml-default", cr.Labels[mlservicev1alpha1.LabelQuota])
	assert.Equal(t, spec.ModelRef, cr.Spec.ModelRef)
	assert.Equal(t, int32(3), cr.Spec.Roles[0].Replicas)
}

func TestServiceToCR_BadJSON_ReturnsError(t *testing.T) {
	row := &Service{
		ID:        uuid.New(),
		Namespace: "team-b",
		Name:      "bad",
		Spec:      datatypes.JSON([]byte(`{"backend":`)),
	}
	_, err := ToCR(row)
	assert.Error(t, err)
}
