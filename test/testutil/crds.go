package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitForCRDEstablished blocks until the named CRD's "Established" condition
// is True or timeout elapses. Useful immediately after envtest startup when a
// reconciler watches a CRD that envtest installed.
func WaitForCRDEstablished(t *testing.T, ctx context.Context, c client.Client, name string, timeout time.Duration) {
	t.Helper()
	Eventually(t, timeout, DefaultPollInterval, func() error {
		var crd apiextv1.CustomResourceDefinition
		if err := c.Get(ctx, types.NamespacedName{Name: name}, &crd); err != nil {
			return err
		}
		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextv1.Established && cond.Status == apiextv1.ConditionTrue {
				return nil
			}
		}
		return fmt.Errorf("CRD %s not Established yet", name)
	})
}
