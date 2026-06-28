package standalone

import (
	"context"
	"io"
	"strconv"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// streamLogs streams a container's logs as a plain text reader, demultiplexing
// Docker's stdout/stderr framing so the caller sees raw log bytes (matching a
// Kubernetes Pod log stream). The instance is expected to already be verified
// as owned by the workload.
func (r *Runtime) streamLogs(ctx context.Context, instance string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	lo := container.LogsOptions{ShowStdout: true, ShowStderr: true}
	if opts != nil {
		lo.Follow = opts.Follow
		lo.Timestamps = opts.Timestamps
		if opts.TailLines != nil {
			lo.Tail = strconv.FormatInt(*opts.TailLines, 10)
		}
	}
	rc, err := r.cli.ContainerLogs(ctx, instance, lo)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil, notFound("pods", types.NamespacedName{Name: instance})
		}
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		_, copyErr := stdcopy.StdCopy(pw, pw, rc)
		_ = rc.Close()
		_ = pw.CloseWithError(copyErr)
	}()
	return pr, nil
}
