// Package compview projects compute-service pod/event wire types into the
// Platform contract Pod/Event types. Shared by the jobs/experiments/mlservices/
// workspaces modules.
package compview

import (
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// Pods projects compute pods into the contract PodList.
func Pods(pods []computeservice.Pod) server.PodList {
	items := make([]server.Pod, 0, len(pods))
	for i := range pods {
		p := &pods[i]
		item := server.Pod{Name: p.Name, Phase: server.PodPhase(p.Phase)}
		if p.NodeName != nil {
			item.NodeName = *p.NodeName
		}
		if p.Labels != nil {
			if role, ok := (*p.Labels)["compute.axisml.io/role"]; ok {
				item.Role = role
			}
		}
		items = append(items, item)
	}
	return server.PodList{Items: items, Count: len(items)}
}

// MetricSeries projects a compute metric series into the contract MetricSeries.
func MetricSeries(m *computeservice.MetricSeries) server.MetricSeries {
	out := server.MetricSeries{
		Metric: m.Metric,
		Range:  m.Range,
		Series: make([]server.MetricPoint, 0, len(m.Series)),
	}
	if m.Step != nil {
		out.Step = *m.Step
	}
	if m.Unit != nil {
		out.Unit = *m.Unit
	}
	for _, p := range m.Series {
		out.Series = append(out.Series, server.MetricPoint{Timestamp: p.Timestamp, Value: p.Value})
	}
	return out
}

// Events projects compute events into the contract EventList.
func Events(events []computeservice.Event) server.EventList {
	items := make([]server.Event, 0, len(events))
	for i := range events {
		e := &events[i]
		item := server.Event{
			Type:   server.EventType(e.Type),
			Reason: e.Reason,
		}
		if e.Note != nil {
			item.Message = *e.Note
		}
		if e.ReportingController != nil {
			item.Source = *e.ReportingController
		}
		// e.EventTime is modelled as an opaque map by the generator, so the
		// structured timestamp is not recoverable here; left zero.
		item.InvolvedObject.Name = e.Object
		items = append(items, item)
	}
	return server.EventList{Items: items, Count: len(items)}
}
