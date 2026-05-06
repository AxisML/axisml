package repo

import (
	"time"

	"github.com/google/uuid"

	"github.com/axisml/axisml/components/artifacts/internal/dbjson"
)

// View is the JSON projection of a Repo returned by the API.
type View struct {
	ID               uuid.UUID         `json:"id"`
	TenantName       string            `json:"tenant_name"`
	Kind             string            `json:"kind"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"display_name,omitempty"`
	Description      string            `json:"description,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	OwnerUser        string            `json:"owner_user,omitempty"`
	Status           string            `json:"status"`
	LatestArtifactID *uuid.UUID        `json:"latest_artifact_id,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func toView(row *Repo) View {
	v := View{
		ID:               row.ID,
		Kind:             row.Kind,
		Name:             row.Name,
		DisplayName:      row.DisplayName,
		Description:      row.Description,
		OwnerUser:        row.OwnerUser,
		Status:           row.Status,
		LatestArtifactID: row.LatestArtifactID,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		Labels:           dbjson.DecodeStringMap(row.Labels),
		Annotations:      dbjson.DecodeStringMap(row.Annotations),
	}
	if row.TenantName != nil {
		v.TenantName = *row.TenantName
	}
	return v
}
