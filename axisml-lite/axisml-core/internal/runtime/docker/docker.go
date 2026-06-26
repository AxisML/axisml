package docker

import (
	"context"
	"fmt"
	"io"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
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

// pullImage pulls an image and drains the progress stream. Best-effort: a pull
// failure surfaces to the caller, which rolls back the apply.
func (r *Runtime) pullImage(ctx context.Context, ref string) error {
	rc, err := r.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	defer func() { _ = rc.Close() }()
	_, _ = io.Copy(io.Discard, rc)
	return nil
}

// createAndStart creates a container from the plan and starts it. It returns
// the container ID so the caller can roll back on a later failure.
func (r *Runtime) createAndStart(ctx context.Context, p *ContainerPlan) (string, error) {
	cfg, hostCfg, netCfg := p.toDocker(r.cfg.WorkloadsNetwork)
	created, err := r.cli.ContainerCreate(ctx, cfg, hostCfg, netCfg, nil, p.Name)
	if err != nil {
		return "", fmt.Errorf("create container %q: %w", p.Name, err)
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
