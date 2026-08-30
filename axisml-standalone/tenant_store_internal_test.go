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

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

func newTenantStoreForTest(t *testing.T) *persistentTenantStore {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&tenantRecord{}))
	return newPersistentTenantStore(db)
}

func persistentTenant(name, cpu string) *tenantv1alpha1.Tenant {
	return &tenantv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: tenantv1alpha1.TenantSpec{
			Namespace: tenantv1alpha1.NamespaceSpec{Name: name},
			Quotas: []tenantv1alpha1.QuotaSpec{{
				Pool: "default",
				Max:  corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)},
			}},
		},
	}
}

func TestPersistentTenantStoreCRUDAndQuotaResolution(t *testing.T) {
	ctx := context.Background()
	store := newTenantStoreForTest(t)

	created := persistentTenant("team-a", "2")
	created.Labels = map[string]string{"team": "e2e"}
	require.NoError(t, store.Create(ctx, created))
	assert.Equal(t, "1", created.ResourceVersion)
	assert.False(t, created.CreationTimestamp.IsZero())

	err := store.Create(ctx, persistentTenant("team-a", "2"))
	assert.True(t, apierrors.IsAlreadyExists(err))

	got, err := store.Get(ctx, "team-a")
	require.NoError(t, err)
	base := got.DeepCopy()
	got.Labels["stage"] = "updated"
	require.NoError(t, store.Patch(ctx, got, base))
	assert.Equal(t, "2", got.ResourceVersion)
	assert.Equal(t, int64(1), got.Generation, "metadata-only patch must not advance generation")

	quotaBase := got.DeepCopy()
	got.Spec.Quotas[0].Max[corev1.ResourceCPU] = resource.MustParse("3")
	require.NoError(t, store.Patch(ctx, got, quotaBase))
	assert.Equal(t, "3", got.ResourceVersion)
	assert.Equal(t, int64(2), got.Generation, "spec patch must advance generation")

	stale := base.DeepCopy()
	stale.Labels["stale"] = "true"
	err = store.Patch(ctx, stale, base)
	assert.True(t, apierrors.IsConflict(err), "want optimistic conflict, got %v", err)

	max, err := store.ResolveQuota(ctx, "team-a", "default")
	require.NoError(t, err)
	assert.Equal(t, int64(3), max.Cpu().Value())

	list, err := store.List(ctx, metav1.ListOptions{LabelSelector: "team=e2e"})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "team-a", list.Items[0].Name)

	require.NoError(t, store.Delete(ctx, "team-a"))
	_, err = store.Get(ctx, "team-a")
	assert.True(t, apierrors.IsNotFound(err))
	assert.True(t, apierrors.IsNotFound(store.Delete(ctx, "team-a")))

	// Kubernetes hard-delete permits explicit name reuse.
	recreated := persistentTenant("team-a", "3")
	require.NoError(t, store.Create(ctx, recreated))
	max, err = store.ResolveQuota(ctx, "team-a", "default")
	require.NoError(t, err)
	assert.Equal(t, int64(3), max.Cpu().Value())
}

func TestPersistentTenantStoreSeedDeletionAndPagination(t *testing.T) {
	ctx := context.Background()
	store := newTenantStoreForTest(t)

	seed := persistentTenant("default", "10")
	require.NoError(t, store.Seed(ctx, seed))
	require.NoError(t, store.Seed(ctx, seed))
	list, err := store.List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, list.Items, 1)

	// A deleted bootstrap tenant is not silently revived on restart/seeding.
	require.NoError(t, store.Delete(ctx, "default"))
	require.NoError(t, store.Seed(ctx, seed))
	_, err = store.Get(ctx, "default")
	assert.True(t, apierrors.IsNotFound(err))

	for _, name := range []string{"team-c", "team-a", "team-b"} {
		require.NoError(t, store.Create(ctx, persistentTenant(name, "1")))
	}
	first, err := store.List(ctx, metav1.ListOptions{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	assert.Equal(t, "team-a", first.Items[0].Name)
	assert.Equal(t, "team-b", first.Items[1].Name)
	assert.NotEmpty(t, first.Continue)

	second, err := store.List(ctx, metav1.ListOptions{Limit: 2, Continue: first.Continue})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.Equal(t, "team-c", second.Items[0].Name)
	assert.Empty(t, second.Continue)
}

func TestPersistentTenantStoreRejectsUnsupportedQuotaResources(t *testing.T) {
	store := newTenantStoreForTest(t)
	tenant := persistentTenant("team-a", "1")
	tenant.Spec.Quotas[0].Max = corev1.ResourceList{"gpu": resource.MustParse("1")}

	err := store.Create(context.Background(), tenant)
	assert.True(t, apierrors.IsInvalid(err), "want Invalid, got %v", err)
	assert.Contains(t, err.Error(), `resource "gpu" is not supported`)
}
