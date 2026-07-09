package gc

import (
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	artmod "github.com/axisml/axisml/axisml-system/artifact-hub/internal/artifact"
	"github.com/axisml/axisml/axisml-system/artifact-hub/internal/artifact/handler"
)

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time { return f.t }

func TestGCArtifact_Maps(t *testing.T) {
	row := artmod.GCRow{
		Namespace: "team-a",
		Kind:      "model",
		Name:      "llama-7b",
		Version:   "v1",
		Digest:    "sha256:abc",
	}
	got := gcArtifact(row)
	assert.Equal(t, handler.Artifact{
		Kind:      "model",
		Namespace: "team-a",
		Name:      "llama-7b",
		Version:   "v1",
		Digest:    "sha256:abc",
	}, got)
	assert.Nil(t, got.Spec, "gcArtifact deliberately leaves Spec unpopulated")
}

func TestRealClock_Now(t *testing.T) {
	now := realClock{}.Now()
	assert.Equal(t, time.UTC, now.Location(), "realClock must report UTC")
	assert.WithinDuration(t, time.Now(), now, 2*time.Second)
}

func TestNew_DefaultsToRealClock(t *testing.T) {
	w := New(5*time.Minute, 24*time.Hour, nil, logr.Discard())
	require.NotNil(t, w)
	_, ok := w.clock.(realClock)
	assert.True(t, ok, "New must install realClock by default")
}

func TestSetClock_ReplacesClock(t *testing.T) {
	w := New(5*time.Minute, 24*time.Hour, nil, logr.Discard())
	fc := fakeClock{t: time.Unix(1000, 0).UTC()}
	w.SetClock(fc)
	assert.Equal(t, fc, w.clock, "SetClock must replace the worker clock")
}
