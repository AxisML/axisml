package docker

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

type fakeImageEngine struct {
	inspectErr   error
	pullErr      error
	pullResponse string
	pullOptions  []image.PullOptions
	inspectCalls int
	pullCalls    int
}

func (f *fakeImageEngine) ImageInspect(
	context.Context,
	string,
	...client.ImageInspectOption,
) (image.InspectResponse, error) {
	f.inspectCalls++
	return image.InspectResponse{}, f.inspectErr
}

func (f *fakeImageEngine) ImagePull(
	_ context.Context,
	_ string,
	options image.PullOptions,
) (io.ReadCloser, error) {
	f.pullCalls++
	f.pullOptions = append(f.pullOptions, options)
	if f.pullErr != nil {
		return nil, f.pullErr
	}
	return io.NopCloser(strings.NewReader(f.pullResponse)), nil
}

func TestEnsureImage(t *testing.T) {
	t.Run("Always pulls without inspecting", func(t *testing.T) {
		cli := &fakeImageEngine{pullResponse: `{"status":"Image is up to date"}` + "\n"}
		require.NoError(t, ensureImage(context.Background(), cli, "busybox:latest", corev1.PullAlways, nil))
		assert.Zero(t, cli.inspectCalls)
		assert.Equal(t, 1, cli.pullCalls)
	})

	t.Run("Always marks pull failures terminal", func(t *testing.T) {
		cli := &fakeImageEngine{
			pullResponse: `{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}` + "\n",
		}
		err := ensureImage(context.Background(), cli, "example.invalid/missing:latest", corev1.PullAlways, nil)
		require.ErrorContains(t, err, "manifest unknown")
		assert.True(t, extensions.IsTerminalApplyError(err))
	})

	t.Run("IfNotPresent uses a local image", func(t *testing.T) {
		cli := &fakeImageEngine{}
		require.NoError(t, ensureImage(context.Background(), cli, "busybox:1.37", corev1.PullIfNotPresent, nil))
		assert.Equal(t, 1, cli.inspectCalls)
		assert.Zero(t, cli.pullCalls)
	})

	t.Run("IfNotPresent pulls a missing image", func(t *testing.T) {
		cli := &fakeImageEngine{
			inspectErr:   cerrdefs.ErrNotFound,
			pullResponse: `{"status":"Downloaded newer image"}` + "\n",
		}
		require.NoError(t, ensureImage(context.Background(), cli, "busybox:1.37", corev1.PullIfNotPresent, nil))
		assert.Equal(t, 1, cli.inspectCalls)
		assert.Equal(t, 1, cli.pullCalls)
	})

	t.Run("IfNotPresent propagates inspect failures", func(t *testing.T) {
		cli := &fakeImageEngine{inspectErr: errors.New("daemon unavailable")}
		err := ensureImage(context.Background(), cli, "busybox:1.37", corev1.PullIfNotPresent, nil)
		require.ErrorContains(t, err, "inspect image")
		assert.False(t, extensions.IsTerminalApplyError(err))
		assert.Zero(t, cli.pullCalls)
	})

	t.Run("Never uses a local image", func(t *testing.T) {
		cli := &fakeImageEngine{}
		require.NoError(t, ensureImage(context.Background(), cli, "busybox:1.37", corev1.PullNever, nil))
		assert.Equal(t, 1, cli.inspectCalls)
		assert.Zero(t, cli.pullCalls)
	})

	t.Run("Never rejects a missing image without pulling", func(t *testing.T) {
		cli := &fakeImageEngine{inspectErr: cerrdefs.ErrNotFound}
		err := ensureImage(context.Background(), cli, "busybox:1.37", corev1.PullNever, nil)
		require.ErrorContains(t, err, "not present locally")
		assert.True(t, extensions.IsTerminalApplyError(err))
		assert.Zero(t, cli.pullCalls)
	})
}

func TestPullImageReadsDockerStreamErrors(t *testing.T) {
	cli := &fakeImageEngine{
		pullResponse: `{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}` + "\n",
	}
	err := pullImage(context.Background(), cli, "example.invalid/missing:latest", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "manifest unknown")
}

func TestPullImagePropagatesRequestErrors(t *testing.T) {
	cli := &fakeImageEngine{pullErr: errors.New("registry unavailable")}
	err := pullImage(context.Background(), cli, "busybox:latest", nil)
	require.ErrorContains(t, err, "registry unavailable")
}

func TestPullImagePassesResolvedRegistryAuth(t *testing.T) {
	cli := &fakeImageEngine{pullResponse: `{"status":"Downloaded"}` + "\n"}
	resolveAuth := func(ref string) (string, error) {
		assert.Equal(t, "registry.example.com/team/model:v1", ref)
		return "encoded-auth", nil
	}

	err := pullImage(
		context.Background(),
		cli,
		"registry.example.com/team/model:v1",
		resolveAuth,
	)
	require.NoError(t, err)
	require.Len(t, cli.pullOptions, 1)
	assert.Equal(t, "encoded-auth", cli.pullOptions[0].RegistryAuth)
}

func TestPullImageDoesNotCallEngineWhenAuthResolutionFails(t *testing.T) {
	cli := &fakeImageEngine{}
	resolveAuth := func(string) (string, error) {
		return "", errors.New("cannot read config")
	}

	err := pullImage(context.Background(), cli, "registry.example.com/model:v1", resolveAuth)
	require.ErrorContains(t, err, "resolve registry credentials")
	require.ErrorContains(t, err, "cannot read config")
	assert.Zero(t, cli.pullCalls)
}
