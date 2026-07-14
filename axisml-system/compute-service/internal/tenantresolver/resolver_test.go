package tenantresolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
)

func TestResolveNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, tenantv1alpha1.AddToScheme(scheme))
	tenant := &tenantv1alpha1.Tenant{}
	tenant.Name = "team-a"
	tenant.Spec.Namespace.Name = "shared-workloads"
	reader := New(fake.NewClientBuilder().WithScheme(scheme).WithObjects(tenant).Build())

	got, err := reader.ResolveNamespace(context.Background(), "team-a")
	require.NoError(t, err)
	assert.Equal(t, "shared-workloads", got)
}

func TestResolveNamespaceMissingTenantUsesLogicalName(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, tenantv1alpha1.AddToScheme(scheme))
	reader := New(fake.NewClientBuilder().WithScheme(scheme).Build())

	got, err := reader.ResolveNamespace(context.Background(), "team-a")
	require.NoError(t, err)
	assert.Equal(t, "team-a", got)
}
