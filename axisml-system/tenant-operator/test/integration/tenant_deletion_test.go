//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	axisml "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	"github.com/axisml/axisml/test/testutil"
)

// TestTenant_DeletionRetainsNamespace verifies the operator's deletion
// semantics: deleting a Tenant CR removes the CR but DELIBERATELY retains the
// namespace (design §6.1: "never delete, no ownerReference" — user workloads and
// data survive an accidental Tenant deletion). Ported from the e2e suite, which
// now runs black-box only; this hermetic envtest is the proper home. The
// orphaned namespace is cleaned up by the test itself.
func TestTenant_DeletionRetainsNamespace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := testEnv.Client

	const (
		tenantName = "team-del"
		tenantNs   = "team-del-ns"
	)
	t.Cleanup(func() {
		// The operator never deletes the namespace, so the test runner does.
		_ = c.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: tenantNs}})
	})

	tenant := &axisml.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantName,
			Labels: map[string]string{axisml.LabelTenantID: "uuid-team-del"},
		},
		Spec: axisml.TenantSpec{
			Namespace: axisml.NamespaceSpec{Name: tenantNs},
		},
	}
	require.NoError(t, c.Create(ctx, tenant))

	// The operator provisions the namespace.
	var ns corev1.Namespace
	testutil.EventuallyExists(t, ctx, c, types.NamespacedName{Name: tenantNs}, &ns, testWaitTimeout)

	// Delete the Tenant CR.
	require.NoError(t, c.Delete(ctx, tenant))

	// The Tenant CR is gone...
	testutil.Eventually(t, testWaitTimeout, 200*time.Millisecond, func() error {
		var got axisml.Tenant
		err := c.Get(ctx, types.NamespacedName{Name: tenantName}, &got)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("tenant %s still present", tenantName)
	})

	// ...but the namespace is intentionally retained.
	var after corev1.Namespace
	require.NoError(t, c.Get(ctx, types.NamespacedName{Name: tenantNs}, &after),
		"namespace must survive tenant deletion")
}
