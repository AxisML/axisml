package docker

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	corev1 "k8s.io/api/core/v1"
)

// ContainerPlan is the runtime-internal adapter boundary: the normalized,
// engine-agnostic description of one container the Docker adapter translates
// into an API request. It is the unit-test object for CR → container rendering
// (design §6.1).
type ContainerPlan struct {
	Name            string
	NamePrefix      string
	Replica         int
	StableOrdinal   bool
	Image           string
	ImagePullPolicy corev1.PullPolicy
	Command         []string // entrypoint
	Args            []string // cmd
	Env             []string // KEY=VALUE
	WorkingDir      string
	Labels          map[string]string
	Ports           []PortPlan
	Mounts          []MountPlan
	Resources       ResourcePlan
	RestartPolicy   string // "no" | "unless-stopped"
	Healthcheck     *HealthPlan
}

// resolveImagePullPolicy applies the Kubernetes defaulting rules locally. Standalone
// receives the API types directly rather than through Kubernetes admission, so
// an omitted policy would otherwise remain empty.
func resolveImagePullPolicy(imageRef string, policy corev1.PullPolicy) (corev1.PullPolicy, error) {
	switch policy {
	case corev1.PullAlways, corev1.PullIfNotPresent, corev1.PullNever:
		return policy, nil
	case "":
		// Default below based on the image reference.
	default:
		return "", capabilityError("unsupported imagePullPolicy %q", policy)
	}

	named, err := reference.ParseNormalizedNamed(imageRef)
	if err != nil {
		return "", capabilityError("invalid image reference %q: %v", imageRef, err)
	}
	if _, ok := named.(reference.Digested); ok {
		return corev1.PullIfNotPresent, nil
	}
	if tagged, ok := named.(reference.Tagged); ok {
		if tagged.Tag() == "latest" {
			return corev1.PullAlways, nil
		}
		return corev1.PullIfNotPresent, nil
	}
	return corev1.PullAlways, nil
}

// PortPlan is a container port projection.
type PortPlan struct {
	Name          string
	ContainerPort int32
	Protocol      string // "tcp" | "udp"
}

// MountPlan is a volume or bind mount. SubPath, when set, mounts only that
// subdirectory of the source (the volume root or host directory) at Target
// instead of the whole volume — so several runs can share one volume and each
// see a different subtree. It is always a cleaned, relative path (no leading
// "/" and no ".." escape); volumeMounts validates it before constructing the
// plan. ConfigMapPath and ConfigMapFiles carry the desired file projection for
// reconciliation; the Docker adapter uses the ordinary Source/SubPath fields.
type MountPlan struct {
	Type           string // "volume" | "bind"
	Source         string
	Target         string
	SubPath        string
	ReadOnly       bool
	ConfigMapPath  string
	ConfigMapFiles map[string]ConfigMapFile
}

// ResourcePlan carries the cgroup limits + GPU request derived from the spec.
// GPUDeviceIDs is the concrete set of physical card indices the allocator pinned
// for this container; it is assigned at admission (not at render) and therefore
// deliberately excluded from the spec hash (see planIdentity).
type ResourcePlan struct {
	NanoCPUs     int64
	MemoryBytes  int64
	GPUCount     int
	GPUDeviceIDs []string
}

// HealthPlan is an optional container healthcheck.
type HealthPlan struct {
	Test     []string
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// toDocker translates the plan into Docker create arguments.
func (p *ContainerPlan) toDocker(net string) (*container.Config, *container.HostConfig, *network.NetworkingConfig) {
	cfg := &container.Config{
		Image:      p.Image,
		Env:        p.Env,
		WorkingDir: p.WorkingDir,
		Labels:     p.Labels,
	}
	if len(p.Command) > 0 {
		cfg.Entrypoint = p.Command
	}
	if len(p.Args) > 0 {
		cfg.Cmd = p.Args
	}
	if len(p.Ports) > 0 {
		cfg.ExposedPorts = nat.PortSet{}
		for _, port := range p.Ports {
			cfg.ExposedPorts[dockerPort(port)] = struct{}{}
		}
	}
	if p.Healthcheck != nil {
		cfg.Healthcheck = &container.HealthConfig{
			Test:     p.Healthcheck.Test,
			Interval: p.Healthcheck.Interval,
			Timeout:  p.Healthcheck.Timeout,
			Retries:  p.Healthcheck.Retries,
		}
	}

	host := &container.HostConfig{
		NetworkMode:   container.NetworkMode(net),
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(p.RestartPolicy)},
		Resources: container.Resources{
			NanoCPUs: p.Resources.NanoCPUs,
			Memory:   p.Resources.MemoryBytes,
		},
	}
	switch {
	case len(p.Resources.GPUDeviceIDs) > 0:
		// Managed mode: pinned to specific cards the allocator reserved.
		host.DeviceRequests = []container.DeviceRequest{{
			Driver:       "nvidia",
			DeviceIDs:    p.Resources.GPUDeviceIDs,
			Capabilities: [][]string{{"gpu"}},
		}}
	case p.Resources.GPUCount > 0:
		// Unmanaged mode (gpu.devices unset): Docker's default count-based
		// request — the NVIDIA runtime picks the cards.
		host.DeviceRequests = []container.DeviceRequest{{
			Driver:       "nvidia",
			Count:        p.Resources.GPUCount,
			Capabilities: [][]string{{"gpu"}},
		}}
	}
	for _, m := range p.Mounts {
		if m.Type == "bind" && m.SubPath != "" && !m.ReadOnly {
			// HostConfig.Binds has the daemon-side source-creation semantics of
			// docker -v. The daemon creates a missing source when the container
			// starts, which also works on engines that advertise API 1.42+ but do
			// not honour Mount.BindOptions.CreateMountpoint. Pass only the resolved
			// subtree so the workload never sees the registered hostPath root.
			source := path.Join(m.Source, m.SubPath)
			host.Binds = append(host.Binds, fmt.Sprintf("%s:%s", source, m.Target))
			continue
		}
		dm := mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		}
		if m.SubPath != "" {
			// A bind mount has no subpath option, so resolve it into the source
			// host path; a named volume carries it as a VolumeOptions.Subpath so
			// Docker mounts the subtree of the shared volume.
			switch m.Type {
			case "bind":
				dm.Source = path.Join(m.Source, m.SubPath)
			case "volume":
				dm.VolumeOptions = &mount.VolumeOptions{Subpath: m.SubPath}
			}
		}
		host.Mounts = append(host.Mounts, dm)
	}

	endpoint := &network.EndpointSettings{}
	if p.Name != "" {
		// Envoy Gateway v1.7 validates Backend FQDNs and rejects single-label
		// Docker container names. Keep the ordinary container-name DNS entry and
		// add a stable multi-label alias for Gateway Backend endpoints.
		endpoint.Aliases = []string{gatewayBackendHostname(p.Name)}
	}
	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{net: endpoint},
	}
	return cfg, host, netCfg
}

func dockerPort(p PortPlan) nat.Port {
	proto := p.Protocol
	if proto == "" {
		proto = "tcp"
	}
	port, _ := nat.NewPort(proto, strconv.Itoa(int(p.ContainerPort)))
	return port
}

// resourcePlan derives cgroup limits + GPU count from a Pod template's limits.
func resourcePlan(rr corev1.ResourceRequirements) ResourcePlan {
	var rp ResourcePlan
	limits := rr.Limits
	if cpu, ok := limits[corev1.ResourceCPU]; ok {
		rp.NanoCPUs = cpu.MilliValue() * 1_000_000
	}
	if mem, ok := limits[corev1.ResourceMemory]; ok {
		rp.MemoryBytes = mem.Value()
	}
	if gpu, ok := limits["nvidia.com/gpu"]; ok {
		rp.GPUCount = int(gpu.Value())
	}
	return rp
}

func formatLabelInt(i int) string { return strconv.Itoa(i) }

func capabilityError(format string, args ...any) error {
	return &CapabilityError{Msg: fmt.Sprintf(format, args...)}
}

// CapabilityError marks a request that names a field or backend standalone does not
// support. The Compute layer maps it to 409 CapabilityUnavailable.
type CapabilityError struct{ Msg string }

func (e *CapabilityError) Error() string { return e.Msg }
