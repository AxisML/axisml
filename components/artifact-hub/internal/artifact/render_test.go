package artifact

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestToView_PopulatesAllJSONFields(t *testing.T) {
	now := time.Now().UTC()
	row := &Artifact{
		ID:          uuid.New(),
		Namespace:   "team-a",
		Kind:        "model",
		Name:        "llama",
		Version:     "v1",
		DisplayName: "Llama",
		Description: "demo",
		OwnerUser:   "alice",
		Spec:        datatypes.JSON([]byte(`{"framework":"pytorch","format":"safetensors"}`)),
		Labels:      datatypes.JSON([]byte(`{"team":"ml"}`)),
		Annotations: datatypes.JSON([]byte(`{"src":"hf"}`)),
		Status:      StatusReady,
		Digest:      "sha256:abc",
		ReadyAt:     &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	v := toView(row)

	assert.Equal(t, "team-a", v.Namespace)
	assert.Equal(t, "model", v.Kind)
	assert.Equal(t, "llama", v.Name)
	assert.Equal(t, "v1", v.Version)
	assert.Equal(t, "Llama", v.DisplayName)
	assert.Equal(t, "demo", v.Description)
	assert.Equal(t, "alice", v.Owner)
	assert.Equal(t, StatusReady, v.Status)
	assert.Equal(t, "sha256:abc", v.Digest)
	assert.Equal(t, "pytorch", v.Spec["framework"])
	assert.Equal(t, "safetensors", v.Spec["format"])
	assert.Equal(t, "ml", v.Labels["team"])
	assert.Equal(t, "hf", v.Annotations["src"])
}

func TestToView_EmptySpecYieldsNilMap(t *testing.T) {
	row := &Artifact{
		ID:        uuid.New(),
		Namespace: "team-b",
		Kind:      "model",
		Name:      "empty",
		Version:   "v1",
		Status:    StatusUploading,
	}
	v := toView(row)
	assert.Nil(t, v.Spec, "empty Spec column must round-trip to nil map (omitempty hides it)")
}

func TestToView_BadSpecJSONIsTolerated(t *testing.T) {
	// toView swallows json.Unmarshal errors so a corrupted row still renders.
	// This is intentional — render must never fail an HTTP response.
	row := &Artifact{
		ID:        uuid.New(),
		Namespace: "team-c",
		Kind:      "model",
		Name:      "bad-json",
		Version:   "v1",
		Spec:      datatypes.JSON([]byte(`{`)), // malformed
		Status:    StatusFailed,
	}
	v := toView(row)
	assert.Equal(t, StatusFailed, v.Status)
}
