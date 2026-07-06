//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtifact_LabelSelectorFilter creates two model versions tagged with
// different axisml.io/project labels and verifies the K8s-style
// labelSelector filters via the artifacts_labels_gin index path.
func TestArtifact_LabelSelectorFilter(t *testing.T) {
	s := setup(t)
	s.resetState(t)

	// Seed alpha and beta artifacts with distinct labels.
	mk := func(name string, labels map[string]string) {
		rr := s.drive(t, http.MethodPost, s.nsPath("/artifacts/"+name), map[string]any{
			"kind":    "model",
			"version": "v1",
			"spec":    map[string]any{"framework": "pytorch", "format": "safetensors"},
			"labels":  labels,
		})
		require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	}
	mk("sel-alpha", map[string]string{"axisml.io/project": "p1"})
	mk("sel-beta", map[string]string{"axisml.io/project": "p2"})

	// labelSelector axisml.io/project=p1 — only sel-alpha
	rr := s.drive(t, http.MethodGet,
		s.nsPath("/artifacts")+"?labelSelector="+url.QueryEscape("axisml.io/project=p1"),
		nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var resp struct {
		Items []map[string]any `json:"items"`
		Count int              `json:"count"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.GreaterOrEqual(t, resp.Count, 1)
	for _, item := range resp.Items {
		require.NotEqual(t, "sel-beta", item["name"], "selector must exclude beta")
	}

	// labelSelector axisml.io/project in (p1,p2) — both rows
	rr = s.drive(t, http.MethodGet,
		s.nsPath("/artifacts")+"?labelSelector="+url.QueryEscape("axisml.io/project in (p1,p2)"),
		nil)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.GreaterOrEqual(t, resp.Count, 2)
}
