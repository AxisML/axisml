package tenant

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// TestToCR_DropsDisplayFields verifies that DisplayName / description /
// labels / annotations from the PG row never appear on the Tenant CR
// (design §6: those stay PG-only).
func TestToCR_DropsDisplayFields(t *testing.T) {
	spec, _ := json.Marshal(SpecJSON{
		Namespace: NamespaceSpec{Name: "team-a-ns"},
	})
	row := &Tenant{
		ID:          uuid.New(),
		Name:        "team-a",
		DisplayName: "Team A (display only)",
		Description: "private",
		Spec:        spec,
	}
	cr, err := ToCR(row)
	require.NoError(t, err)
	assert.Equal(t, "team-a", cr.Name)
	assert.Equal(t, "team-a-ns", cr.Spec.Namespace.Name)
	// The display fields must not have leaked anywhere CR-side.
	assert.NotContains(t, cr.Labels, "display_name")
	assert.NotContains(t, cr.Annotations, "description")
}

// TestToCR_StampsTenantIDLabel ensures the stable orphan-detection anchor
// label is set on the CR.
func TestToCR_StampsTenantIDLabel(t *testing.T) {
	id := uuid.New()
	row := &Tenant{
		ID:   id,
		Name: "team-x",
		Spec: []byte(`{"namespace":{"name":"team-x-ns"}}`),
	}
	cr, err := ToCR(row)
	require.NoError(t, err)
	assert.Equal(t, id.String(), cr.Labels[tenantv1alpha1.LabelTenantID])
}

// TestToCR_RBACPassesThrough ensures ServiceAccount RBAC (rules + roleRef)
// makes it from the PG spec into the rendered Tenant CR.
func TestToCR_RBACPassesThrough(t *testing.T) {
	spec, _ := json.Marshal(SpecJSON{
		Namespace: NamespaceSpec{Name: "rbac-ns"},
		InitResources: &InitResources{
			ServiceAccounts: []ServiceAccountSpec{{
				Name: "ml-worker",
				RBAC: &RBACSpec{
					Rules: []RBACRule{{
						APIGroups: []string{""},
						Resources: []string{"pods", "pods/log"},
						Verbs:     []string{"get", "list", "watch"},
					}},
					RoleRef: &RBACRoleRef{Kind: "ClusterRole", Name: "view"},
				},
			}},
		},
	})
	row := &Tenant{ID: uuid.New(), Name: "rbac-t", Spec: spec}
	cr, err := ToCR(row)
	require.NoError(t, err)
	require.Len(t, cr.Spec.InitResources.ServiceAccounts, 1)
	sa := cr.Spec.InitResources.ServiceAccounts[0]
	require.NotNil(t, sa.RBAC)
	require.Len(t, sa.RBAC.Rules, 1)
	assert.Equal(t, []string{"pods", "pods/log"}, sa.RBAC.Rules[0].Resources)
	require.NotNil(t, sa.RBAC.RoleRef)
	assert.Equal(t, "ClusterRole", sa.RBAC.RoleRef.Kind)
	assert.Equal(t, "view", sa.RBAC.RoleRef.Name)
}
