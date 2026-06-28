package standalone

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// instState is the normalized state of a single managed container, derived from
// a Docker inspect.
type instState struct {
	name       string
	id         string
	image      string
	status     string // created | running | exited | paused | ...
	exitCode   int
	oom        bool
	startedAt  time.Time
	finishedAt time.Time
	health     string // "" | starting | healthy | unhealthy
	role       string
	replica    string
}

// containerName returns a Summary's primary name without Docker's leading "/".
func summaryName(c container.Summary) string {
	if len(c.Names) == 0 {
		return ""
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

// indexByName maps managed containers by their (slash-trimmed) name.
func indexByName(conts []container.Summary) map[string]container.Summary {
	m := make(map[string]container.Summary, len(conts))
	for _, c := range conts {
		m[summaryName(c)] = c
	}
	return m
}

// inspectAll inspects every container and returns its normalized state.
func (r *Runtime) inspectAll(ctx context.Context, conts []container.Summary) ([]instState, error) {
	out := make([]instState, 0, len(conts))
	for _, c := range conts {
		ins, err := r.cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		st := instState{
			name:    summaryName(c),
			id:      c.ID,
			image:   c.Image,
			role:    c.Labels[LabelRole],
			replica: c.Labels[LabelReplicaIndex],
		}
		if ins.State != nil {
			st.status = ins.State.Status
			st.exitCode = ins.State.ExitCode
			st.oom = ins.State.OOMKilled
			st.startedAt = parseDockerTime(ins.State.StartedAt)
			st.finishedAt = parseDockerTime(ins.State.FinishedAt)
			if ins.State.Health != nil {
				st.health = ins.State.Health.Status
			}
		}
		out = append(out, st)
	}
	return out, nil
}

// ready reports whether a container should count as a ready replica: running,
// and (when it declares a healthcheck) healthy. Containers without a
// healthcheck are ready as soon as they run, matching the Kubernetes default
// where a Pod with no readiness probe is Ready when Running.
func (s *instState) ready() bool {
	if s.status != "running" {
		return false
	}
	if s.health == "" {
		return true
	}
	return s.health == "healthy"
}

func parseDockerTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	if t.Year() <= 1 {
		return time.Time{}
	}
	return t.UTC()
}

func notFound(resource string, key types.NamespacedName) error {
	return apierrors.NewNotFound(schema.GroupResource{Group: apiGroup, Resource: resource}, key.Name)
}

const apiGroup = "axisml.io"
