//go:build lite

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/axisml/axisml/test/e2e/internal/clients/artifacthub"
	"github.com/axisml/axisml/test/e2e/internal/clients/clustermanager"
	"github.com/axisml/axisml/test/e2e/internal/clients/computeservice"
)

// liteHarness drives the AxisML Lite deployment form: one axisml-core process
// serving all three System modules at a single base URL, with a fixed default
// tenant and read-only resource pool. It satisfies the same Harness contract as
// the Standard suite, so every CORE test runs against it unchanged.
type liteHarness struct {
	cfg     envConfig
	baseURL string

	clusterManager *clustermanager.ClientWithResponses
	computeService *computeservice.ClientWithResponses
	artifactHub    *artifacthub.ClientWithResponses

	caps map[Capability]bool
}

// Compile-time proof the lite harness satisfies the contract.
var _ Harness = (*liteHarness)(nil)

// h is the global harness handle under the lite build (the Standard build
// declares its own *suite handle).
var h *liteHarness

// liteDefaultTenant is the single static tenant axisml-core serves; multi-tenant
// provisioning is unavailable in Lite (writes return 409 CapabilityUnavailable).
const liteDefaultTenant = "default"

// newHarness builds the Lite harness against LITE_CORE_URL (default
// http://localhost:18080). All three typed clients target the one process; the
// identity header is stamped via the shared request editor.
func newHarness() (*liteHarness, error) {
	cfg := loadConfig()
	base := os.Getenv("LITE_CORE_URL")
	if base == "" {
		base = "http://localhost:18080"
	}
	lh := &liteHarness{cfg: cfg, baseURL: base}

	doer := &http.Client{Timeout: 30 * time.Second}
	user := setUser(cfg.User)
	var err error
	if lh.clusterManager, err = clustermanager.NewClientWithResponses(base,
		clustermanager.WithHTTPClient(doer), clustermanager.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build cluster-manager client: %w", err)
	}
	if lh.computeService, err = computeservice.NewClientWithResponses(base,
		computeservice.WithHTTPClient(doer), computeservice.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build compute-service client: %w", err)
	}
	if lh.artifactHub, err = artifacthub.NewClientWithResponses(base,
		artifacthub.WithHTTPClient(doer), artifacthub.WithRequestEditorFn(user)); err != nil {
		return nil, fmt.Errorf("build artifact-hub client: %w", err)
	}
	return lh, nil
}

func (l *liteHarness) ClusterManager() *clustermanager.ClientWithResponses { return l.clusterManager }
func (l *liteHarness) ComputeService() *computeservice.ClientWithResponses { return l.computeService }
func (l *liteHarness) ArtifactHub() *artifacthub.ClientWithResponses       { return l.artifactHub }

func (l *liteHarness) User() string      { return l.cfg.User }
func (l *liteHarness) config() envConfig { return l.cfg }
func (l *liteHarness) Close()            {}

// Tenant returns the fixed default tenant. Lite has no ElasticQuota enforcement,
// but the compute API still requires a non-empty quota reference on create, so we
// pass the default tenant name (the Standalone runtime ignores it for
// scheduling). CORE tests that assert quota *enforcement* gate on
// CapQuotaEnforcement, which Lite reports false.
func (l *liteHarness) Tenant(*testing.T) (ns, quota string) {
	return liteDefaultTenant, liteDefaultTenant
}

// Ready polls GET /readyz until axisml-core reports serviceable.
func (l *liteHarness) Ready(ctx context.Context) error {
	deadline := time.Now().Add(l.cfg.HTTPReadyTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/readyz", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return l.loadCapabilities(ctx)
			}
			lastErr = fmt.Errorf("GET /readyz: status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(l.cfg.PollInterval)
	}
	return fmt.Errorf("axisml-core at %s not ready within %s: %w", l.baseURL, l.cfg.HTTPReadyTimeout, lastErr)
}

// Supports reads the capability cached at Ready time. Absent capability → false.
func (l *liteHarness) Supports(c Capability) bool { return l.caps[c] }

// loadCapabilities fetches and maps the aggregate GET /api/v1/capabilities
// document axisml-core serves (one per-module sub-document under "components").
func (l *liteHarness) loadCapabilities(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/api/v1/capabilities", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET /api/v1/capabilities: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var doc struct {
		Components map[string]map[string]any `json:"components"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode capabilities: %w", err)
	}
	boolAt := func(component, key string) bool {
		if m := doc.Components[component]; m != nil {
			b, _ := m[key].(bool)
			return b
		}
		return false
	}
	l.caps = map[Capability]bool{
		CapMultiTenant:       boolAt("cluster-manager", "multiTenant"),
		CapResourcePoolWrite: boolAt("cluster-manager", "resourcePoolsWritable"),
		CapQuotaEnforcement:  boolAt("compute-service", "quotaEnforcement"),
		CapArtifactUpload:    boolAt("artifact-hub", "upload"),
	}
	return nil
}
