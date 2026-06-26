//go:build e2e || standard || lite

package e2e

import (
	"os"
	"time"
)

// envConfig holds the knobs the suite reads from the environment. Every value
// has a default that matches a stock `make cluster-up && make helm-install`, so
// a developer can run `make e2e-test` with no extra setup.
type envConfig struct {
	// Namespaces the Helm layers install into.
	InfraNamespace  string
	SystemNamespace string

	// In-cluster Services (name + port) for the three HTTP-surface components.
	// The suite reaches them via `kubectl port-forward` (see harness_test.go).
	ClusterManagerSvc  string
	ClusterManagerPort int
	ComputeServiceSvc  string
	ComputeServicePort int
	ArtifactHubSvc     string
	ArtifactHubPort    int

	// Identity header sent on every HTTP request.
	User string

	// Default ResourcePool + unit that each test file's tenant draws its quota
	// from. A stock install seeds these (Helm hook); the harness reseeds as a
	// fallback. See provisionTenant / ensureDefaultPool.
	DefaultPool string
	DefaultUnit string

	// Workload images, preloaded via `minikube image load` (imagePullPolicy
	// IfNotPresent so the preloaded copies are used; no live pulls mid-run).
	MLRunImage   string
	ServiceImage string

	// Timeout budgets — much larger than the integration layer because real
	// scheduling, image pulls and kubelet startup take real time.
	CRProvisionTimeout   time.Duration
	HTTPReadyTimeout     time.Duration
	PodReadyTimeout      time.Duration
	MLRunCompleteTimeout time.Duration
	PollInterval         time.Duration
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadConfig() envConfig {
	return envConfig{
		InfraNamespace:  envOr("E2E_INFRA_NAMESPACE", "axisml-infra"),
		SystemNamespace: envOr("E2E_SYSTEM_NAMESPACE", "axisml-system"),

		// All three components serve their HTTP API on the `http` port 8080
		// (metrics live on 8081, probes on 8082 — not the API).
		ClusterManagerSvc:  envOr("E2E_CLUSTER_MANAGER_SVC", "axisml-cluster-manager"),
		ClusterManagerPort: 8080,
		ComputeServiceSvc:  envOr("E2E_COMPUTE_SERVICE_SVC", "axisml-compute-service"),
		ComputeServicePort: 8080,
		ArtifactHubSvc:     envOr("E2E_ARTIFACT_HUB_SVC", "axisml-artifact-hub"),
		ArtifactHubPort:    8080,

		User: envOr("E2E_USER", "e2e-suite"),

		DefaultPool: envOr("E2E_DEFAULT_POOL", "default"),
		DefaultUnit: envOr("E2E_DEFAULT_UNIT", "cpu-small"),

		MLRunImage:   envOr("E2E_JOB_IMAGE", "busybox:latest"),
		ServiceImage: envOr("E2E_SERVICE_IMAGE", "nginx:1.27"),

		CRProvisionTimeout:   90 * time.Second,
		HTTPReadyTimeout:     60 * time.Second,
		PodReadyTimeout:      4 * time.Minute,
		MLRunCompleteTimeout: 5 * time.Minute,
		PollInterval:         2 * time.Second,
	}
}
