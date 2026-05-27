package artifact

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/axisml/axisml/components/artifact-hub/internal/dbjson"
)

// View is the JSON projection of an Artifact returned by the API. JSON
// field names follow the design yaml (camelCase) for parity with the
// OpenAPI contract clients consume.
type View struct {
	ID          uuid.UUID         `json:"id"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Visibility  string            `json:"visibility"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Spec        map[string]any    `json:"spec"`
	Status      string            `json:"status"`
	Message     string            `json:"message,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	ReadyAt     *time.Time        `json:"readyAt,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func toView(row *Artifact) View {
	v := View{
		ID:          row.ID,
		Namespace:   row.Namespace,
		Kind:        row.Kind,
		Name:        row.Name,
		Version:     row.Version,
		Visibility:  row.Visibility,
		DisplayName: row.DisplayName,
		Description: row.Description,
		Owner:       row.OwnerUser,
		Status:      row.Status,
		Message:     row.Message,
		Digest:      row.Digest,
		ReadyAt:     row.ReadyAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Labels:      dbjson.DecodeStringMap(row.Labels),
		Annotations: dbjson.DecodeStringMap(row.Annotations),
	}
	if len(row.Spec) > 0 {
		_ = json.Unmarshal(row.Spec, &v.Spec)
	}
	return v
}
