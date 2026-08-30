package serviceadmission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextAdmitsMinimumServingSetThenGrowsIncrementally(t *testing.T) {
	next, roles := Next([]int32{0, 0}, []int32{3, 2})
	assert.Equal(t, []int32{1, 1}, next)
	assert.Equal(t, []int{0, 1}, roles)

	next, roles = Next(next, []int32{3, 2})
	assert.Equal(t, []int32{2, 1}, next)
	assert.Equal(t, []int{0}, roles)

	next, roles = Next([]int32{3, 2}, []int32{3, 2})
	assert.Equal(t, []int32{3, 2}, next)
	require.Empty(t, roles)
}

func TestClampToDesiredOnlyReleasesScaleDown(t *testing.T) {
	assert.Equal(t, []int32{2, 1}, ClampToDesired([]int32{3, 1}, []int32{2, 4}))
}

func TestDecodeDistinguishesLegacyMissingFromPersistedZeroVector(t *testing.T) {
	assert.Equal(t, []int32{3}, Decode(nil, 1, []int32{3}))
	assert.Equal(t, []int32{0}, Decode([]byte(`[]`), 1, []int32{3}))
}
