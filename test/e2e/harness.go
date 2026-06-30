//go:build e2e || standard || lite

package e2e

import (
	"context"
	"testing"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// Harness is the single environment seam between the black-box CORE tests and a
// deployment form. The Standard form backs it with a Kubernetes cluster reached
// over per-service port-forwards; the Lite form backs it with one axisml-core
// process. A CORE test drives only the typed System clients and these methods —
// never a Kubernetes client — so the same test runs unchanged against both forms.
//
// The form is selected at compile time by build tag: harness_select_standard.go
// (e2e||standard) and harness_select_lite.go (lite) each define newHarness.
type Harness interface {
	// Typed, OpenAPI-generated System clients. In Standard each reaches its own
	// service; in Lite all three reach the single axisml-core process. Every call
	// carries the harness identity header (X-Axisml-User).
	ClusterManager() *clustermanager.ClientWithResponses
	ComputeService() *computeservice.ClientWithResponses
	ArtifactHub() *artifacthub.ClientWithResponses

	// Tenant returns a namespace usable for the calling test. Standard provisions
	// a fresh tenant via the cluster-manager API (and returns its ElasticQuota
	// name); Lite returns the fixed default tenant. quota is "" when the form has
	// no ElasticQuota (Lite).
	Tenant(t *testing.T) (ns, quota string)

	// Supports reports whether the form provides a capability, read from the
	// services' /api/v1/capabilities documents. CORE tests gate form-specific
	// behaviour with t.Skip when a capability is absent.
	Supports(c Capability) bool

	// Ready blocks until the form is serviceable (Standard: CRDs established +
	// default pool seeded; Lite: GET /readyz). Called once from TestMain.
	Ready(ctx context.Context) error

	// User is the identity stamped on every request (X-Axisml-User).
	User() string

	// Close tears down any per-run resources (port-forwards, etc.).
	Close()

	// config exposes the shared env-driven knobs (timeouts, poll interval,
	// default pool/unit, workload images). Unexported: only the in-package CORE
	// helpers read it.
	config() envConfig
}

// Capability names a feature whose presence differs by deployment form. The set
// is intentionally small: only capabilities that actually gate a CORE test.
type Capability string

const (
	// CapMultiTenant: the cluster-manager can create tenants (Standard); Lite
	// serves a single static default tenant.
	CapMultiTenant Capability = "multiTenant"
	// CapResourcePoolWrite: the cluster-manager can create/patch/delete resource
	// pools (Standard); Lite serves a single read-only default pool.
	CapResourcePoolWrite Capability = "resourcePoolsWritable"
	// CapQuotaEnforcement: the scheduler admits pods against an ElasticQuota
	// (Standard axisml-scheduler); the Lite Standalone runtime has no quota gate.
	CapQuotaEnforcement Capability = "quotaEnforcement"
	// CapArtifactUpload: two-phase artifact upload is available (both forms).
	CapArtifactUpload Capability = "artifactUpload"
)
