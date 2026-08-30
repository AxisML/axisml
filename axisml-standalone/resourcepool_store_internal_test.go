package standalone

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
)

func newResourcePoolStoreForTest(t *testing.T) *persistentResourcePoolStore {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&resourcePoolRecord{}))
	return newPersistentResourcePoolStore(db)
}

func persistentResourcePool(name, cpu string) *cmv1alpha1.ResourcePool {
	return &cmv1alpha1.ResourcePool{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: cmv1alpha1.ResourcePoolSpec{
			Units: []cmv1alpha1.ResourceUnit{{
				Name:     "small",
				Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)},
				Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)},
			}},
		},
	}
}

func TestPersistentResourcePoolStoreCRUDAndResolution(t *testing.T) {
	ctx := context.Background()
	store := newResourcePoolStoreForTest(t)

	created := persistentResourcePool("team-pool", "2")
	created.Labels = map[string]string{"team": "e2e"}
	require.NoError(t, store.Create(ctx, created))
	assert.Equal(t, "1", created.ResourceVersion)
	assert.False(t, created.CreationTimestamp.IsZero())

	err := store.Create(ctx, persistentResourcePool("team-pool", "2"))
	assert.True(t, apierrors.IsAlreadyExists(err))

	got, err := store.Get(ctx, "team-pool")
	require.NoError(t, err)
	base := got.DeepCopy()
	got.Labels["stage"] = "updated"
	require.NoError(t, store.Patch(ctx, got, base))
	assert.Equal(t, "2", got.ResourceVersion)
	assert.Equal(t, int64(1), got.Generation)

	specBase := got.DeepCopy()
	got.Spec.Units[0].Requests[corev1.ResourceCPU] = resource.MustParse("3")
	require.NoError(t, store.Patch(ctx, got, specBase))
	assert.Equal(t, "3", got.ResourceVersion)
	assert.Equal(t, int64(2), got.Generation)

	stale := base.DeepCopy()
	stale.Labels["stale"] = "true"
	err = store.Patch(ctx, stale, base)
	assert.True(t, apierrors.IsConflict(err), "want optimistic conflict, got %v", err)

	resolved, err := store.ResolveResourcePool(ctx, "team-pool")
	require.NoError(t, err)
	assert.Equal(t, "team-pool", resolved.Name)
	unit, err := store.ResolveResourceUnit(ctx, "team-pool", "small")
	require.NoError(t, err)
	assert.Equal(t, int64(3), unit.Requests.Cpu().Value())

	list, err := store.List(ctx, metav1.ListOptions{LabelSelector: "team=e2e"})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "team-pool", list.Items[0].Name)

	require.NoError(t, store.Delete(ctx, "team-pool"))
	_, err = store.Get(ctx, "team-pool")
	assert.True(t, apierrors.IsNotFound(err))
	assert.True(t, apierrors.IsNotFound(store.Delete(ctx, "team-pool")))

	recreated := persistentResourcePool("team-pool", "4")
	require.NoError(t, store.Create(ctx, recreated))
	unit, err = store.ResolveResourceUnit(ctx, "team-pool", "small")
	require.NoError(t, err)
	assert.Equal(t, int64(4), unit.Requests.Cpu().Value())
}

func TestPersistentResourcePoolStoreSeedDeletionAndPagination(t *testing.T) {
	ctx := context.Background()
	store := newResourcePoolStoreForTest(t)

	seed := persistentResourcePool("default", "10")
	require.NoError(t, store.Seed(ctx, seed))
	require.NoError(t, store.Seed(ctx, seed))
	list, err := store.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)

	require.NoError(t, store.Delete(ctx, "default"))
	require.NoError(t, store.Seed(ctx, seed))
	_, err = store.Get(ctx, "default")
	assert.True(t, apierrors.IsNotFound(err))

	for _, name := range []string{"pool-c", "pool-a", "pool-b"} {
		require.NoError(t, store.Create(ctx, persistentResourcePool(name, "1")))
	}
	first, err := store.List(ctx, metav1.ListOptions{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	assert.Equal(t, "pool-a", first.Items[0].Name)
	assert.Equal(t, "pool-b", first.Items[1].Name)
	assert.NotEmpty(t, first.Continue)

	second, err := store.List(ctx, metav1.ListOptions{Limit: 2, Continue: first.Continue})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.Equal(t, "pool-c", second.Items[0].Name)
	assert.Empty(t, second.Continue)
}

func TestPersistentResourcePoolStoreRejectsKubernetesSchedulingFields(t *testing.T) {
	store := newResourcePoolStoreForTest(t)
	pool := persistentResourcePool("unsupported", "1")
	pool.Spec.NodeSelector = map[string]string{"accelerator": "gpu"}

	err := store.Create(context.Background(), pool)
	assert.True(t, apierrors.IsInvalid(err), "want Invalid, got %v", err)
}

func TestPersistentResourcePoolStoreRejectsUnsupportedResources(t *testing.T) {
	store := newResourcePoolStoreForTest(t)
	pool := persistentResourcePool("unsupported", "1")
	pool.Spec.Units[0].Limits = corev1.ResourceList{"gpu": resource.MustParse("1")}

	err := store.Create(context.Background(), pool)
	assert.True(t, apierrors.IsInvalid(err), "want Invalid, got %v", err)
	assert.Contains(t, err.Error(), `resource "gpu" is not supported`)
}
