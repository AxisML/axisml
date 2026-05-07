//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestService_NotFound covers GET and scale against a service name that
// doesn't exist. DELETE is intentionally idempotent (always 204 — see
// services_lifecycle_test.go for the happy-path delete + reconciler reap).
func TestService_NotFound(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	const ns = "services-nf-ns"
	c, err := client.New(testCfg, client.Options{Scheme: testScheme})
	require.NoError(t, err)
	_ = c.Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	})

	rr := doRequestJSON(t, http.MethodGet, "/api/v1/namespaces/"+ns+"/services/ghost", "")
	require.Equal(t, http.StatusNotFound, rr.Code)

	rr = postJSON(t, "/api/v1/namespaces/"+ns+"/services/ghost/scale",
		map[string]any{"replicas": 2})
	require.Equal(t, http.StatusNotFound, rr.Code)
}

// TestService_ScaleValidation rejects negative replica counts.
func TestService_ScaleValidation(t *testing.T) {
	if testEngine == nil {
		t.Skip("compute test scaffolding not initialised")
	}
	// We don't need an existing service: the bind-time validator (gte=0)
	// fires before the row lookup, so an arbitrary path is sufficient.
	rr := postJSON(t, "/api/v1/namespaces/x-ns/services/whatever/scale",
		map[string]any{"replicas": -1})
	if rr.Code < 400 || rr.Code >= 500 {
		t.Fatalf("expected 4xx, got %d body=%s", rr.Code, rr.Body.String())
	}
}
