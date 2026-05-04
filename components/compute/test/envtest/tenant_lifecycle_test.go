//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tenantv1alpha1 "github.com/axisml/axisml/components/operators/tenant-operator/api/v1alpha1"

	quotamod "github.com/axisml/axisml/components/compute/internal/quota"
	poolmod "github.com/axisml/axisml/components/compute/internal/resourcepool"
	tenantmod "github.com/axisml/axisml/components/compute/internal/tenant"
	"github.com/axisml/axisml/test/testutil"
)

func TestTenantOutboxCreatesCR(t *testing.T) {
	require.NotNil(t, testManager, "manager not bootstrapped")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pools := poolmod.NewService(gormDB)
	quotas := quotamod.NewService(gormDB, pools)
	tenants := tenantmod.NewService(gormDB, quotas, pools)

	created, err := tenants.Create(ctx, tenantmod.CreateInput{
		Name:        "envtest-t1",
		DisplayName: "EnvTest Tenant 1",
		Namespace:   tenantv1alpha1.NamespaceSpec{Name: "envtest-t1"},
	})
	require.NoError(t, err)

	rec := tenantmod.NewReconciler(gormDB, testManager.GetClient(), pools, logr.Discard(), 100*time.Millisecond)
	rec.SetQuotas(quotas)

	// Run reconciler in a short loop until the CR appears.
	go func() { _ = rec.Start(ctx) }()

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Name: "envtest-t1"}, cr, 10*time.Second)
	require.Equal(t, created.Name, cr.Name)
	require.Equal(t, created.Namespace.Name, cr.Spec.Namespace.Name)
	require.Equal(t, created.ID.String(), cr.Labels[tenantv1alpha1.LabelTenantID])
}

func TestTenantInformerReflectsActive(t *testing.T) {
	require.NotNil(t, testManager, "manager not bootstrapped")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pools := poolmod.NewService(gormDB)
	quotas := quotamod.NewService(gormDB, pools)
	tenants := tenantmod.NewService(gormDB, quotas, pools)

	created, err := tenants.Create(ctx, tenantmod.CreateInput{
		Name:      "envtest-t2",
		Namespace: tenantv1alpha1.NamespaceSpec{Name: "envtest-t2"},
	})
	require.NoError(t, err)

	rec := tenantmod.NewReconciler(gormDB, testManager.GetClient(), pools, logr.Discard(), 100*time.Millisecond)
	rec.SetQuotas(quotas)
	go func() { _ = rec.Start(ctx) }()

	informer := tenantmod.NewInformer(gormDB, testManager, quotas, logr.Discard())
	go func() { _ = informer.Start(ctx) }()

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Name: "envtest-t2"}, cr, 10*time.Second)

	// Simulate tenant-operator: write status.phase=Active.
	cr.Status = tenantv1alpha1.TenantStatus{
		Phase:          tenantv1alpha1.TenantPhaseActive,
		NamespaceReady: true,
	}
	require.NoError(t, cl.Status().Update(ctx, cr))

	testutil.Eventually(t, 10*time.Second, testutil.DefaultPollInterval, func() error {
		v, err := tenants.GetByID(ctx, created.ID)
		if err != nil {
			return err
		}
		if v.Status != string(tenantmod.StatusActive) {
			return fmt.Errorf("tenant status=%s, want Active", v.Status)
		}
		return nil
	})
}

// Verify external CR delete (during Deleting status) drives PG to Deleted.
func TestTenantExternalDeleteDuringDeleting(t *testing.T) {
	require.NotNil(t, testManager, "manager not bootstrapped")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pools := poolmod.NewService(gormDB)
	quotas := quotamod.NewService(gormDB, pools)
	tenants := tenantmod.NewService(gormDB, quotas, pools)

	created, err := tenants.Create(ctx, tenantmod.CreateInput{
		Name:      "envtest-t3",
		Namespace: tenantv1alpha1.NamespaceSpec{Name: "envtest-t3"},
	})
	require.NoError(t, err)

	rec := tenantmod.NewReconciler(gormDB, testManager.GetClient(), pools, logr.Discard(), 100*time.Millisecond)
	rec.SetQuotas(quotas)
	go func() { _ = rec.Start(ctx) }()

	informer := tenantmod.NewInformer(gormDB, testManager, quotas, logr.Discard())
	go func() { _ = informer.Start(ctx) }()

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	// Wait for CR.
	cr := &tenantv1alpha1.Tenant{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Name: "envtest-t3"}, cr, 10*time.Second)

	// Move PG to Deleting first.
	require.NoError(t, tenants.Delete(ctx, "envtest-t3"))

	// Wait for reconciler to delete the CR and informer to push to Deleted.
	testutil.EventuallyGone(t, ctx, cl, client.ObjectKey{Name: "envtest-t3"}, &tenantv1alpha1.Tenant{}, 10*time.Second)

	testutil.Eventually(t, 10*time.Second, testutil.DefaultPollInterval, func() error {
		v, err := tenants.GetByID(ctx, created.ID)
		if err != nil {
			return err
		}
		if v.Status != string(tenantmod.StatusDeleted) {
			return fmt.Errorf("tenant status=%s, want Deleted", v.Status)
		}
		return nil
	})

	// Compile guards so unused imports don't lint-fail.
	_ = metav1.ObjectMeta{}
}
