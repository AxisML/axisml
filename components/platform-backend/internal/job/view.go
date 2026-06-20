package job

import (
	"encoding/json"

	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
)

// LabelJob is the grouping label that ties a Run (MLRun) to its Job.
const LabelJob = "axisml.io/job"

func toView(d *store.Definition) server.Job {
	var spec server.JobSpec
	_ = json.Unmarshal([]byte(specJSON(d.Spec)), &spec)
	return server.Job{
		ID:          server.UUID(d.ID),
		Namespace:   d.TenantName,
		TenantName:  d.TenantName,
		Name:        d.Name,
		DisplayName: d.DisplayName,
		Description: d.Description,
		Owner:       d.OwnerUser,
		Labels:      server.StringMap(d.Labels),
		Annotations: server.StringMap(d.Annotations),
		Spec:        spec,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func specJSON(j store.JSONB) string {
	if len(j) == 0 {
		return "{}"
	}
	return string(j)
}

func marshalSpec(spec server.JobSpec) store.JSONB {
	b, err := json.Marshal(spec)
	if err != nil {
		return store.JSONB("{}")
	}
	return store.JSONB(b)
}

func jsonUnmarshalSpec(j store.JSONB, out *server.JobSpec) error {
	return json.Unmarshal([]byte(specJSON(j)), out)
}
