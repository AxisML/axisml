//go:build envtest

package envtest_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mljobv1alpha1 "github.com/axisml/axisml/components/operator/api/mljob/v1alpha1"
	mlservicev1alpha1 "github.com/axisml/axisml/components/operator/api/mlservice/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/components/operator/api/tenant/v1alpha1"

	"github.com/axisml/axisml/test/testutil"
)

// apiFixture sets up a minimal tenant + pool + unit + quota chain via the
// HTTP API so that job / service tests can dispatch CR creates immediately.
// All fixture rows use names parameterised on the test's stem so concurrent
// tests don't collide.
type apiFixture struct {
	TenantName string
	TenantNS   string
	PoolName   string
	UnitName   string
	UnitID     string
	QuotaName  string
	QuotaID    string
}

func setupAPIFixture(t *testing.T, ctx context.Context, stem string) apiFixture {
	t.Helper()
	fx := apiFixture{
		TenantName: stem + "-tenant",
		TenantNS:   stem + "-ns",
		PoolName:   stem + "-pool",
		UnitName:   stem + "-small",
		QuotaName:  stem + "-quota",
	}
	t.Cleanup(func() {
		// Cascade delete in dependency-reverse order. We ignore errors since
		// some rows may have already been removed by the test body.
		_ = doJSON(t, context.Background(), http.MethodDelete,
			pathf("/api/v1/tenants/%s/quotas/%s", fx.TenantName, fx.QuotaName), nil, nil)
		_ = doJSON(t, context.Background(), http.MethodDelete,
			pathf("/api/v1/resource-pools/%s/resource-units/%s", fx.PoolName, fx.UnitName), nil, nil)
		_ = doJSON(t, context.Background(), http.MethodDelete,
			pathf("/api/v1/resource-pools/%s", fx.PoolName), nil, nil)
		_ = doJSON(t, context.Background(), http.MethodDelete,
			pathf("/api/v1/tenants/%s", fx.TenantName), nil, nil)
	})

	// Pre-create the tenant namespace. In production, the tenant-operator
	// reconciles the Tenant CR into a Namespace; envtest doesn't run the
	// tenant-operator (compute envtest only loads its CRDs), so the
	// namespace must exist before child CRs (MLJob/MLService) can land.
	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	if err := cl.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: fx.TenantNS},
	}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("create tenant namespace: %v", err)
	}

	// Tenant.
	rr := doJSON(t, ctx, http.MethodPost, "/api/v1/tenants", map[string]any{
		"name":      fx.TenantName,
		"namespace": map[string]any{"name": fx.TenantNS},
	}, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Pool.
	rr = doJSON(t, ctx, http.MethodPost, "/api/v1/resource-pools", map[string]any{
		"name": fx.PoolName,
	}, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Unit (carries CPU/memory the job/service injects into its Pod template).
	rr = doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/resource-pools/%s/resource-units", fx.PoolName),
		map[string]any{
			"name":     fx.UnitName,
			"requests": map[string]any{"cpu": "100m", "memory": "64Mi"},
			"limits":   map[string]any{"cpu": "200m", "memory": "128Mi"},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)
	fx.UnitID, _ = idAndName(t, rr)

	// Quota.
	rr = doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/quotas", fx.TenantName),
		map[string]any{
			"pool": fx.PoolName,
			"name": fx.QuotaName,
			"max":  map[string]any{"cpu": "4", "memory": "8Gi"},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)
	fx.QuotaID, _ = idAndName(t, rr)

	return fx
}

// TestJobsAPI_LifecycleCreatesAndCancelsCR covers the full job HTTP flow:
//
//  1. POST /jobs creates the DB row.
//  2. The job reconciler turns that into an MLJob CR (env-tested at L1 by
//     the existing tenant_lifecycle scenarios for the same machinery).
//  3. POST /jobs/:job/cancel transitions DB status to Canceling and the
//     reconciler patches CR.spec.runPolicy.suspend=true.
//  4. DELETE /jobs/:job transitions DB to Deleting and the CR is removed.
func TestJobsAPI_LifecycleCreatesAndCancelsCR(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := setupAPIFixture(t, ctx, "api-job")

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	jobName := "hello"

	// Create.
	rr := doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/jobs", fx.TenantName),
		map[string]any{
			"name":           jobName,
			"resourceUnitId": fx.UnitID,
			"quotaId":        fx.QuotaID,
			"roles": []map[string]any{{
				"name":     mljobv1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{
					"image":   "busybox:latest",
					"command": []string{"sh", "-c", "echo hello"},
				},
			}},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)

	// Reconciler creates the MLJob CR.
	cr := &mljobv1alpha1.MLJob{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Namespace: fx.TenantNS, Name: jobName}, cr, 15*time.Second)
	require.NotEmpty(t, cr.Labels[mljobv1alpha1.LabelJobID], "compute must stamp job-id label")
	require.NotEmpty(t, cr.Labels[mljobv1alpha1.LabelQuota],
		"compute must stamp quota label — mljob-operator validation rejects empty")
	require.False(t, cr.Spec.RunPolicy.Suspend, "fresh CR should not be suspended")

	// Simulate the mljob-operator advancing the CR to Pending — envtest has
	// no operator running, so the informer would otherwise never see a
	// non-empty phase and the DB would stay in Creating forever (Cancel
	// rejects 412 in that state).
	patch := client.MergeFrom(cr.DeepCopy())
	cr.Status.Phase = mljobv1alpha1.PhasePending
	require.NoError(t, cl.Status().Patch(ctx, cr, patch))

	// Wait for the informer to advance the DB row out of Creating.
	testutil.Eventually(t, 15*time.Second, testutil.DefaultPollInterval, func() error {
		var view struct {
			Status string `json:"status"`
		}
		grr := doJSON(t, ctx, http.MethodGet,
			pathf("/api/v1/tenants/%s/jobs/%s", fx.TenantName, jobName), nil, &view)
		if grr.Code != http.StatusOK {
			return fmt.Errorf("get job: %s", grr.Body.String())
		}
		if view.Status == "Creating" {
			return fmt.Errorf("job still in Creating; informer hasn't observed CR phase yet")
		}
		return nil
	})

	// Cancel.
	rr = doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/jobs/%s/cancel", fx.TenantName, jobName), nil, nil)
	requireStatus(t, rr, http.StatusOK)

	// CR.spec.runPolicy.suspend flips to true.
	testutil.Eventually(t, 15*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &mljobv1alpha1.MLJob{}
		if err := cl.Get(ctx, client.ObjectKey{Namespace: fx.TenantNS, Name: jobName}, fresh); err != nil {
			return err
		}
		if !fresh.Spec.RunPolicy.Suspend {
			return fmt.Errorf("CR.spec.runPolicy.suspend not yet true")
		}
		return nil
	})

	// Delete (DB → Deleting; reconciler removes the CR).
	rr = doJSON(t, ctx, http.MethodDelete,
		pathf("/api/v1/tenants/%s/jobs/%s", fx.TenantName, jobName), nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	testutil.EventuallyGone(t, ctx, cl, client.ObjectKey{Namespace: fx.TenantNS, Name: jobName}, &mljobv1alpha1.MLJob{}, 15*time.Second)
}

// TestJobsAPI_QuotaMismatchRejected verifies the cross-tenant guard in the
// service layer: a job referencing a quota that belongs to a different
// tenant is rejected with 4xx, not silently 201ed and reconciled forever
// against a non-existent ElasticQuota.
func TestJobsAPI_QuotaMismatchRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fxA := setupAPIFixture(t, ctx, "api-job-a")
	fxB := setupAPIFixture(t, ctx, "api-job-b")

	// Submit a job for tenantA but pass tenantB's quotaID.
	rr := doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/jobs", fxA.TenantName),
		map[string]any{
			"name":           "wrong-quota",
			"resourceUnitId": fxA.UnitID,
			"quotaId":        fxB.QuotaID,
			"roles": []map[string]any{{
				"name":     mljobv1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{"image": "busybox:latest"},
			}},
		}, nil)
	if rr.Code < 400 || rr.Code >= 500 {
		t.Fatalf("expected 4xx for cross-tenant quota; got status=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestServicesAPI_LifecycleCreatesScalesAndDeletes drives the service HTTP
// flow analogous to the job test, plus the /:scale custom action.
func TestServicesAPI_LifecycleCreatesScalesAndDeletes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := setupAPIFixture(t, ctx, "api-svc")

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	svcName := "predictor"

	// Build the create body. The roles[].template echoes mlservice-operator's
	// PodTemplate shape (Image + Ports). Resources come from the resource
	// unit and are injected by compute, so we don't set them here.
	rr := doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/services", fx.TenantName),
		map[string]any{
			"name":           svcName,
			"resourceUnitId": fx.UnitID,
			"quotaId":        fx.QuotaID,
			"modelRef":       map[string]any{"name": "demo", "version": "v1"},
			"roles": []map[string]any{{
				"name":     mlservicev1alpha1.DefaultRoleName,
				"replicas": 1,
				"template": map[string]any{
					"image": "nginx:1.27",
					"ports": []map[string]any{{
						"name":          "http",
						"containerPort": 8080,
						"protocol":      string(corev1.ProtocolTCP),
					}},
				},
			}},
		}, nil)
	requireStatus(t, rr, http.StatusCreated)

	// MLService CR appears with replicas=1.
	cr := &mlservicev1alpha1.MLService{}
	testutil.EventuallyExists(t, ctx, cl, client.ObjectKey{Namespace: fx.TenantNS, Name: svcName}, cr, 15*time.Second)
	require.NotEmpty(t, cr.Labels[mlservicev1alpha1.LabelServiceID])
	require.NotEmpty(t, cr.Labels[mlservicev1alpha1.LabelQuota],
		"compute must stamp quota label — mlservice-operator validation rejects empty")
	require.Len(t, cr.Spec.Roles, 1)
	require.Equal(t, int32(1), cr.Spec.Roles[0].Replicas)

	// Scale to 3 via /:scale.
	rr = doJSON(t, ctx, http.MethodPost,
		pathf("/api/v1/tenants/%s/services/%s/scale", fx.TenantName, svcName),
		map[string]any{"replicas": 3}, nil)
	requireStatus(t, rr, http.StatusOK)

	testutil.Eventually(t, 15*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &mlservicev1alpha1.MLService{}
		if err := cl.Get(ctx, client.ObjectKey{Namespace: fx.TenantNS, Name: svcName}, fresh); err != nil {
			return err
		}
		if len(fresh.Spec.Roles) == 0 || fresh.Spec.Roles[0].Replicas != 3 {
			return fmt.Errorf("CR.spec.roles[0].replicas not yet 3")
		}
		return nil
	})

	// Delete.
	rr = doJSON(t, ctx, http.MethodDelete,
		pathf("/api/v1/tenants/%s/services/%s", fx.TenantName, svcName), nil, nil)
	requireStatus(t, rr, http.StatusNoContent)
	testutil.EventuallyGone(t, ctx, cl, client.ObjectKey{Namespace: fx.TenantNS, Name: svcName}, &mlservicev1alpha1.MLService{}, 15*time.Second)
}

// TestQuotasAPI_UpdatePropagatesToTenantCR verifies the quota PATCH flow
// updates the Tenant CR's spec.quotas via the tenant reconciler. This
// catches the dirty-marking hook (quota.SetTenantDirtyHook) silently
// regressing — without it, quota updates would diverge from the CR.
func TestQuotasAPI_UpdatePropagatesToTenantCR(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fx := setupAPIFixture(t, ctx, "api-quota")

	cl, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)

	// Wait for the Tenant CR to carry the quota set up by the fixture.
	testutil.Eventually(t, 15*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &tenantv1alpha1.Tenant{}
		if err := cl.Get(ctx, client.ObjectKey{Name: fx.TenantName}, fresh); err != nil {
			return err
		}
		for _, q := range fresh.Spec.Quotas {
			if q.Name == fx.QuotaName {
				return nil
			}
		}
		return fmt.Errorf("Tenant CR.spec.quotas does not include fixture quota yet")
	})

	// PATCH max to a higher value.
	rr := doJSON(t, ctx, http.MethodPatch,
		pathf("/api/v1/tenants/%s/quotas/%s", fx.TenantName, fx.QuotaName),
		map[string]any{
			"max": map[string]any{"cpu": "16", "memory": "32Gi"},
		}, nil)
	requireStatus(t, rr, http.StatusOK)

	// Tenant CR's quota max reflects the new value.
	testutil.Eventually(t, 15*time.Second, testutil.DefaultPollInterval, func() error {
		fresh := &tenantv1alpha1.Tenant{}
		if err := cl.Get(ctx, client.ObjectKey{Name: fx.TenantName}, fresh); err != nil {
			return err
		}
		for _, q := range fresh.Spec.Quotas {
			if q.Name != fx.QuotaName {
				continue
			}
			if !q.Max.Cpu().Equal(resource.MustParse("16")) {
				return fmt.Errorf("Tenant CR quota max.cpu not yet 16")
			}
			return nil
		}
		return fmt.Errorf("quota gone from Tenant CR")
	})
}
