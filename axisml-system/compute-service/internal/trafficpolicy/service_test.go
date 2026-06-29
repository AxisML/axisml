package trafficpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mltp "github.com/axisml/axisml/axisml-system/compute-operator/api/mltrafficpolicy/v1alpha1"
)

func members(pairs ...any) []mltp.BackendMember {
	var out []mltp.BackendMember
	for i := 0; i < len(pairs); i += 3 {
		out = append(out, mltp.BackendMember{
			ServiceName: pairs[i].(string),
			Role:        pairs[i+1].(mltp.BackendRole),
			Weight:      int32(pairs[i+2].(int)),
		})
	}
	return out
}

func TestValidateWeights(t *testing.T) {
	// weighted must sum to 100
	require.NoError(t, validateWeights(mltp.TrafficModeWeighted, members("a", mltp.BackendRole(""), 70, "b", mltp.BackendRole(""), 30)))
	assert.Error(t, validateWeights(mltp.TrafficModeWeighted, members("a", mltp.BackendRole(""), 70, "b", mltp.BackendRole(""), 20)))

	// canary must also sum to 100
	require.NoError(t, validateWeights(mltp.TrafficModeCanary, members("a", mltp.RoleStable, 90, "b", mltp.RoleCanary, 10)))

	// out-of-range weight
	assert.Error(t, validateWeights(mltp.TrafficModeWeighted, members("a", mltp.BackendRole(""), 150, "b", mltp.BackendRole(""), -50)))

	// bluegreen all-or-nothing
	require.NoError(t, validateWeights(mltp.TrafficModeBlueGreen, members("blue", mltp.RoleBlue, 100, "green", mltp.RoleGreen, 0)))
	assert.Error(t, validateWeights(mltp.TrafficModeBlueGreen, members("blue", mltp.RoleBlue, 60, "green", mltp.RoleGreen, 40)))
}

func TestValidateModeShape(t *testing.T) {
	// weighted needs >=2
	assert.Error(t, validateModeShape(mltp.TrafficModeWeighted, members("a", mltp.BackendRole(""), 100)))

	// canary needs exactly one stable + one canary
	require.NoError(t, validateModeShape(mltp.TrafficModeCanary, members("a", mltp.RoleStable, 90, "b", mltp.RoleCanary, 10)))
	assert.Error(t, validateModeShape(mltp.TrafficModeCanary, members("a", mltp.RoleStable, 90, "b", mltp.RoleStable, 10)))

	// duplicate member name
	assert.Error(t, validateModeShape(mltp.TrafficModeWeighted, members("a", mltp.BackendRole(""), 50, "a", mltp.BackendRole(""), 50)))
}

func TestRoleIndex(t *testing.T) {
	bs := members("a", mltp.RoleStable, 90, "b", mltp.RoleCanary, 10)
	assert.Equal(t, 0, roleIndex(bs, mltp.RoleStable))
	assert.Equal(t, 1, roleIndex(bs, mltp.RoleCanary))
	assert.Equal(t, -1, roleIndex(bs, mltp.RoleBlue))
}
