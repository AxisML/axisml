// Package artifactdef implements the Models and Images tags: name-level
// definitions (Platform PG) plus their versions (artifact-hub), kind-parameterised
// so both share one implementation (backend.md §4.5). Definitions are tuple-
// addressed (/{kind-plural}/{tenant}/{name}); versions are artifact-hub's
// (namespace, kind, name, version).
package artifactdef

import (
	"encoding/json"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/artifacthub"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/store"
)

func defView(d *store.Definition, kind server.DefinitionKind) server.ArtifactDefinition {
	var spec map[string]any
	_ = json.Unmarshal([]byte(specJSON(d.Spec)), &spec)
	return server.ArtifactDefinition{
		ID:          server.UUID(d.ID),
		Namespace:   d.TenantName,
		TenantName:  d.TenantName,
		Name:        d.Name,
		Kind:        kind,
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

// versionView projects an artifact-hub View into the contract Model shape (Image
// has identical JSON, so the same projection serves both kinds).
func versionView(v *artifacthub.Artifact, tenant string) server.Model {
	out := server.Model{
		ID:          server.UUID(v.Id.String()),
		Namespace:   v.Namespace,
		TenantName:  tenant,
		Name:        v.Name,
		Version:     v.Version,
		DisplayName: strv(v.DisplayName),
		Description: strv(v.Description),
		Status:      server.ModelStatus(v.Status),
		Source:      server.ArtifactSource(strv(v.Source)),
		Digest:      strv(v.Digest),
		Spec:        v.Spec,
		Owner:       strv(v.Owner),
		CreatedAt:   v.CreatedAt,
		ReadyAt:     v.ReadyAt,
		UpdatedAt:   v.UpdatedAt,
	}
	return out
}

func specJSON(j store.JSONB) string {
	if len(j) == 0 {
		return "{}"
	}
	return string(j)
}

func marshalSpec(spec map[string]any) store.JSONB {
	if spec == nil {
		return store.JSONB("{}")
	}
	b, err := json.Marshal(spec)
	if err != nil {
		return store.JSONB("{}")
	}
	return store.JSONB(b)
}

func strv(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
