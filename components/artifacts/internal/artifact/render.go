package artifact

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/axisml/axisml/components/artifacts/internal/dbjson"
)

// View is the JSON projection of an Artifact returned by the API.
type View struct {
	ID          uuid.UUID         `json:"id"`
	Namespace   string            `json:"namespace"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	OwnerUser   string            `json:"owner_user,omitempty"`
	Spec        map[string]any    `json:"spec"`
	Status      string            `json:"status"`
	Message     string            `json:"message,omitempty"`
	Digest      string            `json:"digest,omitempty"`
	ReadyAt     *time.Time        `json:"ready_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func toView(row *Artifact) View {
	v := View{
		ID:          row.ID,
		Namespace:   row.Namespace,
		Kind:        row.Kind,
		Name:        row.Name,
		Version:     row.Version,
		DisplayName: row.DisplayName,
		Description: row.Description,
		OwnerUser:   row.OwnerUser,
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
