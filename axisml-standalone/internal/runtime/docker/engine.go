package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	corev1 "k8s.io/api/core/v1"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// managedFilter builds a label filter that selects this installation's managed
// resources of a given kind for a (namespace, name).
func (r *Runtime) managedFilter(kind, namespace, name string) filters.Args {
	f := filters.NewArgs()
	f.Add("label", LabelManaged+"=true")
	f.Add("label", LabelResourceKind+"="+kind)
	f.Add("label", LabelNamespace+"="+namespace)
	f.Add("label", LabelName+"="+name)
	return f
}

// listContainers returns all (incl. stopped) managed containers for a workload.
func (r *Runtime) listContainers(ctx context.Context, kind, namespace, name string) ([]container.Summary, error) {
	return r.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: r.managedFilter(kind, namespace, name),
	})
}

// ensureNetwork creates the named bridge network if absent. Idempotent.
func (r *Runtime) ensureNetwork(ctx context.Context, name string) error {
	f := filters.NewArgs()
	f.Add("name", name)
	nets, err := r.cli.NetworkList(ctx, network.ListOptions{Filters: f})
	if err != nil {
		return fmt.Errorf("list networks: %w", err)
	}
	for _, n := range nets {
		if n.Name == name {
			return nil
		}
	}
	_, err = r.cli.NetworkCreate(ctx, name, network.CreateOptions{Driver: "bridge"})
	if err != nil {
		return fmt.Errorf("create network %q: %w", name, err)
	}
	return nil
}

// imageEngineClient is the Docker image API subset used by the pull-policy
// implementation. Keeping it narrow makes all policy branches testable without
// a Docker daemon.
type imageEngineClient interface {
	ImageInspect(ctx context.Context, imageID string, inspectOpts ...client.ImageInspectOption) (image.InspectResponse, error)
	ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error)
}

// pullImage pulls an image and consumes Docker's JSON progress stream. Pull
// failures such as authentication and missing manifests are often reported in
// the stream after the HTTP request succeeds, so draining it with io.Copy is
// not sufficient.
func pullImage(
	ctx context.Context,
	cli imageEngineClient,
	ref string,
	resolveAuth registryAuthResolver,
) error {
	options := image.PullOptions{}
	if resolveAuth != nil {
		auth, err := resolveAuth(ref)
		if err != nil {
			return fmt.Errorf("resolve registry credentials for image %q: %w", ref, err)
		}
		options.RegistryAuth = auth
	}
	rc, err := cli.ImagePull(ctx, ref, options)
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	defer func() { _ = rc.Close() }()
	dec := json.NewDecoder(rc)
	for {
		var msg struct {
			Error       string `json:"error"`
			ErrorDetail *struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read pull response for image %q: %w", ref, err)
		}
		if msg.ErrorDetail != nil && msg.ErrorDetail.Message != "" {
			return fmt.Errorf("pull image %q: %s", ref, msg.ErrorDetail.Message)
		}
		if msg.Error != "" {
			return fmt.Errorf("pull image %q: %s", ref, msg.Error)
		}
	}
}

// ensureImage enforces Kubernetes-compatible image pull semantics before a
// container is created.
func ensureImage(
	ctx context.Context,
	cli imageEngineClient,
	ref string,
	policy corev1.PullPolicy,
	resolveAuth registryAuthResolver,
) error {
	switch policy {
	case corev1.PullAlways:
		return extensions.NewTerminalApplyError(pullImage(ctx, cli, ref, resolveAuth))
	case corev1.PullIfNotPresent:
		if _, err := cli.ImageInspect(ctx, ref); err == nil {
			return nil
		} else if !cerrdefs.IsNotFound(err) {
			return fmt.Errorf("inspect image %q: %w", ref, err)
		}
		return extensions.NewTerminalApplyError(pullImage(ctx, cli, ref, resolveAuth))
	case corev1.PullNever:
		if _, err := cli.ImageInspect(ctx, ref); err != nil {
			if cerrdefs.IsNotFound(err) {
				return extensions.NewTerminalApplyError(
					fmt.Errorf("image %q is not present locally and imagePullPolicy is Never", ref),
				)
			}
			return fmt.Errorf("inspect image %q: %w", ref, err)
		}
		return nil
	default:
		return extensions.NewTerminalApplyError(
			fmt.Errorf("unsupported imagePullPolicy %q for image %q", policy, ref),
		)
	}
}

// createAndStart creates a container from the plan and starts it. It returns
// the container ID so the caller can roll back on a later failure.
func (r *Runtime) createAndStart(ctx context.Context, p *ContainerPlan) (string, error) {
	cfg, hostCfg, netCfg := p.toDocker(r.cfg.WorkloadsNetwork)
	var created container.CreateResponse
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		created, err = r.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, p.Name)
		if err == nil {
			break
		}
		if !cerrdefs.IsConflict(err) || p.StableOrdinal || p.NamePrefix == "" {
			return "", fmt.Errorf("create container %q: %w", p.Name, err)
		}
		p.Name = instanceName(p.NamePrefix, p.Replica, false)
	}
	if err != nil {
		return "", fmt.Errorf("create container %q after suffix retries: %w", p.Name, err)
	}
	if err := r.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		_ = r.cli.ContainerRemove(ctx, created.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("start container %q: %w", p.Name, err)
	}
	return created.ID, nil
}

// removeContainer force-removes a container by id. A missing container is not
// an error.
func (r *Runtime) removeContainer(ctx context.Context, id string) error {
	err := r.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true, RemoveVolumes: false})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return err
	}
	return nil
}

// stopContainer stops a container with the default timeout. A missing or
// already-stopped container is not an error.
func (r *Runtime) stopContainer(ctx context.Context, id string) error {
	err := r.cli.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return err
	}
	return nil
}

// ensureVolume creates a named managed volume if absent. Idempotent.
func (r *Runtime) ensureVolume(ctx context.Context, name string, labels map[string]string) error {
	_, err := r.cli.VolumeCreate(ctx, volume.CreateOptions{Name: name, Labels: labels})
	return err
}

// removeVolume removes a named volume. A missing volume is not an error.
func (r *Runtime) removeVolume(ctx context.Context, name string) error {
	err := r.cli.VolumeRemove(ctx, name, true)
	if err != nil && !cerrdefs.IsNotFound(err) {
		return err
	}
	return nil
}
