package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	cmext "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"

	"github.com/axisml/axisml/axisml-lite/axisml-core/pkg/core"
)

func TestConfigResourceCatalog_Reads(t *testing.T) {
	ctx := context.Background()
	cat := core.NewConfigResourceCatalog(validPool())

	got, err := cat.Get(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, "default", got.Name)

	_, err = cat.Get(ctx, "nope")
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))

	list, err := cat.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "default", list.Items[0].Name)
}

func TestConfigResourceCatalog_WritesUnavailable(t *testing.T) {
	ctx := context.Background()
	cat := core.NewConfigResourceCatalog(validPool())

	assert.False(t, cat.Writable())
	assert.ErrorIs(t, cat.Create(ctx, &cmv1alpha1.ResourcePool{}), cmext.ErrCapabilityUnavailable)
	assert.ErrorIs(t, cat.Patch(ctx, &cmv1alpha1.ResourcePool{}, &cmv1alpha1.ResourcePool{}), cmext.ErrCapabilityUnavailable)
	assert.ErrorIs(t, cat.Delete(ctx, "default"), cmext.ErrCapabilityUnavailable)
}

func TestConfigResourceCatalog_ResolvePool(t *testing.T) {
	ctx := context.Background()
	cat := core.NewConfigResourceCatalog(validPool())

	got, err := cat.ResolveResourcePool(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, "default", got.Name)

	_, err = cat.ResolveResourcePool(ctx, "")
	require.Error(t, err)

	_, err = cat.ResolveResourcePool(ctx, "other")
	require.Error(t, err)
}

func TestConfigResourceCatalog_ResolveUnit(t *testing.T) {
	ctx := context.Background()
	cat := core.NewConfigResourceCatalog(validPool())

	unit, err := cat.ResolveResourceUnit(ctx, "default", "small")
	require.NoError(t, err)
	assert.Equal(t, "small", unit.Name)

	_, err = cat.ResolveResourceUnit(ctx, "default", "")
	require.Error(t, err)

	_, err = cat.ResolveResourceUnit(ctx, "default", "missing")
	require.Error(t, err)

	// Unknown pool fails at the pool-resolution step.
	_, err = cat.ResolveResourceUnit(ctx, "other", "small")
	require.Error(t, err)
}

func TestStaticTenantStore_Reads(t *testing.T) {
	ctx := context.Background()
	store := core.NewStaticTenantStore(validTenant())

	got, err := store.Get(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, "default", got.Name)

	_, err = store.Get(ctx, "nope")
	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))

	list, err := store.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "default", list.Items[0].Name)
}

func TestStaticTenantStore_Writes(t *testing.T) {
	ctx := context.Background()
	store := core.NewStaticTenantStore(validTenant())

	// Create is idempotent for the default tenant only.
	require.NoError(t, store.Create(ctx, &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "default"}}))
	assert.ErrorIs(t, store.Create(ctx, &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "other"}}), cmext.ErrCapabilityUnavailable)
	assert.ErrorIs(t, store.Create(ctx, nil), cmext.ErrCapabilityUnavailable)

	assert.ErrorIs(t, store.Patch(ctx, &tenantv1alpha1.Tenant{}, &tenantv1alpha1.Tenant{}), cmext.ErrCapabilityUnavailable)
	assert.ErrorIs(t, store.Delete(ctx, "default"), cmext.ErrCapabilityUnavailable)
	assert.False(t, store.Writable())
}

func TestConfigResourceCatalog_MultiplePools(t *testing.T) {
	ctx := context.Background()
	gpu := validPool()
	gpu.Name = "gpu"
	cat := core.NewConfigResourceCatalog(validPool(), gpu)

	list, err := cat.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 2)

	got, err := cat.Get(ctx, "gpu")
	require.NoError(t, err)
	assert.Equal(t, "gpu", got.Name)

	resolved, err := cat.ResolveResourcePool(ctx, "default")
	require.NoError(t, err)
	assert.Equal(t, "default", resolved.Name)
}

func TestStaticTenantStore_MultipleTenants(t *testing.T) {
	ctx := context.Background()
	teamA := validTenant()
	teamA.Name = "team-a"
	teamA.Spec.Namespace.Name = "team-a"
	store := core.NewStaticTenantStore(validTenant(), teamA)

	list, err := store.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 2)

	got, err := store.Get(ctx, "team-a")
	require.NoError(t, err)
	assert.Equal(t, "team-a", got.Name)

	// Create is idempotent for any configured tenant; unknown names are unavailable.
	require.NoError(t, store.Create(ctx, &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "team-a"}}))
	assert.ErrorIs(t, store.Create(ctx, &tenantv1alpha1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "team-b"}}), cmext.ErrCapabilityUnavailable)
}

// Guard against the sentinel drifting into a wrapped/renamed error.
func TestErrCapabilityUnavailableIsSentinel(t *testing.T) {
	err := core.NewConfigResourceCatalog(validPool()).Create(context.Background(), &cmv1alpha1.ResourcePool{})
	assert.True(t, errors.Is(err, cmext.ErrCapabilityUnavailable))
}
