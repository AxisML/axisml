package compview_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/compview"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

func strPtr(s string) *string { return &s }

func TestPods(t *testing.T) {
	node := "node-1"
	labels := map[string]string{"compute.axisml.io/role": "worker", "other": "x"}
	labelsNoRole := map[string]string{"other": "x"}

	tests := []struct {
		name     string
		in       []computeservice.Pod
		wantNode string
		wantRole string
	}{
		{
			name:     "node and role set",
			in:       []computeservice.Pod{{Name: "p1", Phase: "Running", NodeName: &node, Labels: &labels}},
			wantNode: "node-1",
			wantRole: "worker",
		},
		{
			name:     "nil node, nil labels",
			in:       []computeservice.Pod{{Name: "p2", Phase: "Pending"}},
			wantNode: "",
			wantRole: "",
		},
		{
			name:     "labels present without role key",
			in:       []computeservice.Pod{{Name: "p3", Phase: "Running", Labels: &labelsNoRole}},
			wantNode: "",
			wantRole: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compview.Pods(tt.in)
			require.Len(t, got.Items, 1)
			assert.Equal(t, 1, got.Count)
			assert.Equal(t, tt.in[0].Name, got.Items[0].Name)
			assert.Equal(t, server.PodPhase(tt.in[0].Phase), got.Items[0].Phase)
			assert.Equal(t, tt.wantNode, got.Items[0].NodeName)
			assert.Equal(t, tt.wantRole, got.Items[0].Role)
		})
	}
}

func TestPods_Empty(t *testing.T) {
	got := compview.Pods(nil)
	assert.Empty(t, got.Items)
	assert.Equal(t, 0, got.Count)
}

func TestEvents(t *testing.T) {
	tests := []struct {
		name        string
		in          []computeservice.Event
		wantMessage string
		wantSource  string
		wantObject  string
	}{
		{
			name: "note and controller set",
			in: []computeservice.Event{{
				Type:                "Normal",
				Reason:              "Scheduled",
				Note:                strPtr("pod scheduled"),
				ReportingController: strPtr("scheduler"),
				Object:              "Pod/p1",
			}},
			wantMessage: "pod scheduled",
			wantSource:  "scheduler",
			wantObject:  "Pod/p1",
		},
		{
			name: "nil note and controller",
			in: []computeservice.Event{{
				Type:   "Warning",
				Reason: "FailedScheduling",
				Object: "Pod/p2",
			}},
			wantMessage: "",
			wantSource:  "",
			wantObject:  "Pod/p2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compview.Events(tt.in)
			require.Len(t, got.Items, 1)
			assert.Equal(t, 1, got.Count)
			assert.Equal(t, server.EventType(tt.in[0].Type), got.Items[0].Type)
			assert.Equal(t, tt.in[0].Reason, got.Items[0].Reason)
			assert.Equal(t, tt.wantMessage, got.Items[0].Message)
			assert.Equal(t, tt.wantSource, got.Items[0].Source)
			assert.Equal(t, tt.wantObject, got.Items[0].InvolvedObject.Name)
		})
	}
}

func TestEvents_Empty(t *testing.T) {
	got := compview.Events(nil)
	assert.Empty(t, got.Items)
	assert.Equal(t, 0, got.Count)
}
