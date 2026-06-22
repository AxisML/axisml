package experiment

import (
	"encoding/json"

	"github.com/axisml/axisml/components/platform/internal/clients/computeservice"
	"github.com/axisml/axisml/components/platform/internal/server"
	"github.com/axisml/axisml/components/platform/internal/store"
)

// LabelExperiment ties a Run (MLRun) to its Experiment.
const LabelExperiment = "axisml.io/experiment"

func toView(d *store.Definition) server.Experiment {
	var spec server.JobSpec
	_ = json.Unmarshal([]byte(specJSON(d.Spec)), &spec)
	return server.Experiment{
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

// tbToView projects a compute tensorboard MLService into the contract TensorBoard.
func tbToView(s *computeservice.MLService) server.TensorBoard {
	v := server.TensorBoard{
		Name:      s.Name,
		Phase:     server.TensorBoardPhase(s.Phase),
		CreatedAt: s.CreatedAt,
	}
	// endpoint URL + message live in the service status (best-effort extract).
	var st struct {
		Message  string `json:"message"`
		Endpoint string `json:"endpoint"`
	}
	if b, err := json.Marshal(s.Status); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	v.URL = st.Endpoint
	v.Message = st.Message
	return v
}
