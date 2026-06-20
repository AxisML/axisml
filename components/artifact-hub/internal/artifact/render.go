package artifact

import (
	"encoding/json"

	"github.com/axisml/axisml/components/artifact-hub/internal/dbjson"
	"github.com/axisml/axisml/components/artifact-hub/internal/server"
	"github.com/axisml/axisml/components/artifact-hub/internal/store"
)

func toView(row *store.Artifact) server.Artifact {
	v := server.Artifact{
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
		DeletedAt:   row.DeletedAt,
		Labels:      dbjson.DecodeStringMap(row.Labels),
		Annotations: dbjson.DecodeStringMap(row.Annotations),
	}
	if len(row.Spec) > 0 {
		_ = json.Unmarshal(row.Spec, &v.Spec)
	}
	return v
}
