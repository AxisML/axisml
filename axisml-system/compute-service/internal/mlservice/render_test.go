package mlservice

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	mlservicev1alpha1 "github.com/axisml/axisml/axisml-system/compute-operator/api/mlservice/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
)

func TestMLServiceToCR_StampsLabelsAndCopiesSpec(t *testing.T) {
	id := uuid.New()
	spec := mlservicev1alpha1.MLServiceSpec{
		Backend:    mlservicev1alpha1.Backend{Name: "native", Engine: "deployment"},
		Scheduling: mlservicev1alpha1.Scheduling{Quota: "axisml-default"},
		Roles: []mlservicev1alpha1.RoleSpec{{
			Name:     mlservicev1alpha1.DefaultRoleName,
			Replicas: 3,
		}},
	}
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)

	row := &store.MLService{
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
	assert.Equal(t, int32(3), cr.Spec.Roles[0].Replicas)
}

func TestMLServiceToCR_BadJSON_ReturnsError(t *testing.T) {
	row := &store.MLService{
		ID:        uuid.New(),
		Namespace: "team-b",
		Name:      "bad",
		Spec:      datatypes.JSON([]byte(`{"backend":`)),
	}
	_, err := ToCR(row)
	assert.Error(t, err)
}
