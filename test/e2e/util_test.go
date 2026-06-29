//go:build (e2e || standard) && !lite

package e2e

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tenantv1 "github.com/axisml/axisml/axisml-system/tenant-operator/api/v1alpha1"
)

// removeTenant best-effort cleans up a Standard tenant: soft-delete via the API,
// then hard-remove the CR + namespace via the admin client (the operator never
// deletes the namespace itself). Used by provisionTenant's cleanup.
func removeTenant(name, ns string) {
	bg := context.Background()
	_, _ = h.deleteTenant(bg, name)
	_ = h.k8s.Delete(bg, &tenantv1.Tenant{ObjectMeta: metav1.ObjectMeta{Name: name}})
	_ = h.k8s.Delete(bg, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
}
