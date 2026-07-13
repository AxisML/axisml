package standalone

import (
	"fmt"
	"strconv"
	"time"

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
	Name          string
	Image         string
	Command       []string // entrypoint
	Args          []string // cmd
	Env           []string // KEY=VALUE
	WorkingDir    string
	Labels        map[string]string
	Ports         []PortPlan
	Mounts        []MountPlan
	Resources     ResourcePlan
	RestartPolicy string // "no" | "unless-stopped"
	Healthcheck   *HealthPlan
}

// PortPlan is a container port projection.
type PortPlan struct {
	Name          string
	ContainerPort int32
	Protocol      string // "tcp" | "udp"
}

// MountPlan is a volume or bind mount.
type MountPlan struct {
	Type     string // "volume" | "bind"
	Source   string
	Target   string
	ReadOnly bool
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
		// Unmanaged mode (AXISML_GPU_DEVICES unset): Docker's default count-based
		// request — the NVIDIA runtime picks the cards.
		host.DeviceRequests = []container.DeviceRequest{{
			Driver:       "nvidia",
			Count:        p.Resources.GPUCount,
			Capabilities: [][]string{{"gpu"}},
		}}
	}
	for _, m := range p.Mounts {
		host.Mounts = append(host.Mounts, mount.Mount{
			Type:     mount.Type(m.Type),
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	netCfg := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{net: {}},
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

// envToStrings converts the curated EnvVar subset to KEY=VALUE strings. ValueFrom
// references are unsupported in Lite and reported as a capability error.
func envToStrings(env []corev1.EnvVar) ([]string, error) {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if e.ValueFrom != nil {
			return nil, capabilityError("env[%s].valueFrom is unsupported in Lite", e.Name)
		}
		out = append(out, e.Name+"="+e.Value)
	}
	return out, nil
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

// CapabilityError marks a request that names a field or backend Lite does not
// support. The Compute layer maps it to 409 CapabilityUnavailable.
type CapabilityError struct{ Msg string }

func (e *CapabilityError) Error() string { return e.Msg }
