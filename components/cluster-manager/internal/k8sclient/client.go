// Package k8sclient builds a controller-runtime client for the
// cluster-manager. The client is bypass-cache (always direct API server
// reads) so a freshly-created Tenant CR is immediately visible to the
// next GET — important for the "create then poll status" flow.
package k8sclient

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	tenantv1alpha1 "github.com/axisml/axisml/components/tenant-operator/api/v1alpha1"
)

// Build constructs a direct (cache-less) client.Client wired with the
// Tenant CRD scheme.
func Build() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	return BuildWithConfig(cfg)
}

// BuildWithConfig is the test-friendly variant that accepts an explicit
// rest.Config (e.g., from envtest).
func BuildWithConfig(cfg *rest.Config) (client.Client, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(tenantv1alpha1.AddToScheme(scheme))

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}
	return c, nil
}
