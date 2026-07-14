// Package standalone is the in-process Standalone Runtime: the single-host
// implementation of the published extensions.ComputeRuntime contract used by
// AxisML Lite. It lives under internal/runtime/ — the home for runtime
// backends. It receives the same MLRun / MLService / MLTrafficPolicy desired
// objects the Kubernetes runtime does and maps them onto Docker containers,
// volumes, networks and Traefik dynamic config, then reports status back as the
// corresponding CR Status.
//
// This adapter is the only code in axisml-core that touches the Docker socket.
// The Compute domain layer drives it purely through the ComputeRuntime
// interface, exactly as it drives the Kubernetes runtime (design §3.1, §4).
package standalone

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/axisml/axisml/axisml-system/apis/pkg/workloadname"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// Managed-resource label keys (design §5.2). The resource key
// (kind, namespace, name) plus LabelManaged is the authoritative identity of
// every container the runtime creates.
const (
	LabelManaged      = "io.axisml.managed"
	LabelResourceKind = "io.axisml.resource-kind"
	LabelNamespace    = "io.axisml.resource-namespace"
	LabelName         = "io.axisml.resource-name"
	LabelReplicaIndex = "io.axisml.replica-index"
	LabelRole         = "compute.axisml.io/role"
	LabelTenant       = "tenant.axisml.io/name"
	LabelSpecHash     = "io.axisml.spec-hash"
	LabelGPUDevices   = "io.axisml.gpu-devices"

	KindRun     = "run"
	KindService = "service"
	KindTraffic = "traffic-policy"
)

// Config configures the Standalone runtime adapter.
type Config struct {
	// Network dynamic workloads join (Traefik also joins it to route them).
	WorkloadsNetwork string
	// TraefikDir is the Traefik file-provider dynamic config directory.
	TraefikDir string
	// HostPathVolumes maps a predefined data volume's name (= the claim name a
	// workload mounts) to a host directory. A workload that mounts such a volume
	// by name gets the host path bind-mounted instead of a managed Docker volume.
	// Seeded from Tenant.spec.initResources.volumes[] entries that set hostPath.
	HostPathVolumes map[string]string
	// GPUDevices is the set of physical GPU indices the host has handed to
	// AxisML for scheduling. Empty disables GPU scheduling (GPU workloads stay
	// Pending). Resolved from AXISML_GPU_DEVICES via ResolveGPUDevices.
	GPUDevices []int
}

// Runtime implements extensions.ComputeRuntime over the Docker Engine API.
type Runtime struct {
	cli    *client.Client
	cfg    Config
	log    logr.Logger
	events *eventRing

	// cancelled tracks Runs whose containers were stopped by an explicit
	// CancelMLRun, so Observe can surface Suspended=CancelRequested even though
	// a stopped container is otherwise indistinguishable from a failure.
	mu        sync.Mutex
	cancelled map[string]bool

	// gpu serialises GPU admission and holds the schedulable device set.
	gpu *gpuAllocator
}

var _ extensions.ComputeRuntime = (*Runtime)(nil)

// New builds a Runtime from an existing Docker client.
func New(cli *client.Client, cfg Config, log logr.Logger) *Runtime {
	return &Runtime{
		cli:       cli,
		cfg:       cfg,
		log:       log,
		events:    newEventRing(512),
		cancelled: map[string]bool{},
		gpu:       newGPUAllocator(cfg.GPUDevices),
	}
}

// NewClient builds a Docker client with API-version negotiation. host may be
// empty to use the SDK default (DOCKER_HOST or the unix socket).
func NewClient(host string) (*client.Client, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if host != "" {
		opts = append(opts, client.WithHost(host))
	}
	return client.NewClientWithOpts(opts...)
}

// EnsureNetwork creates the workloads network if it is absent. Idempotent.
func (r *Runtime) EnsureNetwork(ctx context.Context) error {
	return r.ensureNetwork(ctx, r.cfg.WorkloadsNetwork)
}

func (r *Runtime) cancelKey(namespace, name string) string {
	return namespace + "/" + name
}

func (r *Runtime) markCancelled(namespace, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancelled[r.cancelKey(namespace, name)] = true
}

func (r *Runtime) isCancelled(namespace, name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancelled[r.cancelKey(namespace, name)]
}

func (r *Runtime) clearCancelled(namespace, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancelled, r.cancelKey(namespace, name))
}

// baseLabels returns the managed labels common to every resource of a kind. The
// tenant label mirrors the workload namespace: in Lite the tenant scope IS the
// namespace (design §5.2).
func (r *Runtime) baseLabels(kind, namespace, name string) map[string]string {
	return map[string]string{
		LabelManaged:      "true",
		LabelResourceKind: kind,
		LabelNamespace:    namespace,
		LabelName:         name,
		LabelTenant:       namespace,
	}
}

var nameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// instanceName mirrors Kubernetes-generated instance naming. Jobs and
// Deployments get a fresh five-character suffix for each instance lifecycle;
// StatefulSet instances keep their ordinal.
func instanceName(base string, replica int, stableOrdinal bool) string {
	raw := base + "-" + utilrand.String(5)
	if stableOrdinal {
		raw = fmt.Sprintf("%s-%d", base, replica)
	}
	clean := nameSanitizer.ReplaceAllString(raw, "-")
	if clean == raw && len(clean) <= 63 {
		return clean
	}
	h := shortHash(raw)
	trimmed := clean
	if len(trimmed) > 54 {
		trimmed = trimmed[:54]
	}
	return fmt.Sprintf("%s-%s", strings.TrimRight(trimmed, "-"), h)
}

func instanceBase(obj metav1.Object, role string) string {
	return workloadname.Role(obj, role)
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

// specHash hashes a plan's identity-relevant fields so Apply can skip a rebuild
// when nothing changed (design §7.1).
func specHash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16]
}

func nowMetaTime() time.Time { return time.Now().UTC() }
