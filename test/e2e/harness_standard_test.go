//go:build (e2e || standard) && !lite

package e2e

import (
	"context"
	"testing"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// This file adapts the Standard suite (a real Kubernetes cluster reached over
// port-forwards) to the Harness interface the CORE tests drive. The suite owns
// all the Kubernetes machinery; these methods expose only the black-box surface.

// Compile-time proof the suite satisfies the harness contract.
var _ Harness = (*suite)(nil)

// newHarness builds the Standard harness: a K8s client over the ambient
// kubeconfig plus the three port-forwarded System clients. Selected under the
// standard build tag (harness_select_lite_test.go provides the Lite variant).
func newHarness() (*suite, error) { return newSuite() }

func (s *suite) ClusterManager() *clustermanager.ClientWithResponses { return s.clusterManager }
func (s *suite) ComputeService() *computeservice.ClientWithResponses { return s.computeService }
func (s *suite) ArtifactHub() *artifacthub.ClientWithResponses       { return s.artifactHub }

func (s *suite) User() string      { return s.cfg.User }
func (s *suite) config() envConfig { return s.cfg }

// Close tears down the port-forwards, including the platform-layer forward the
// platform tests start lazily (it is not owned by the suite's forward list).
func (s *suite) Close() {
	if platformPF != nil {
		platformPF.Stop()
	}
	s.close()
}

// Ready runs the Standard fail-fast gate: required CRDs Established + the default
// ResourcePool seeded. The port-forwards proved HTTP reachability already.
func (s *suite) Ready(ctx context.Context) error { return gateReady(ctx) }

// Tenant provisions a fresh multi-tenant scope via the cluster-manager API and
// returns its namespace plus the koord ElasticQuota name workloads schedule under.
func (s *suite) Tenant(t *testing.T) (ns, quota string) { return provisionTenant(t) }

// Supports reports Standard capabilities. The full Kubernetes form backs every
// capability (multi-tenant, writable pools, koord quota enforcement, artifact
// upload), so it reports true across the board — matching the per-service
// /api/v1/capabilities documents.
func (s *suite) Supports(Capability) bool { return true }
