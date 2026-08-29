package docker

import (
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// eventRing is a fixed-capacity in-memory ring of runtime events. Events live
// for the current process lifetime only (design §6.6); the events API returns
// recent apply/cancel/delete results and inspect-driven transitions.
type eventRing struct {
	mu  sync.Mutex
	cap int
	buf []eventRec
}

type eventRec struct {
	kind      string
	namespace string
	name      string
	instance  string // "" = resource-level
	reason    string
	message   string
	at        time.Time
}

func newEventRing(capacity int) *eventRing {
	return &eventRing{cap: capacity, buf: make([]eventRec, 0, capacity)}
}

func (e *eventRing) record(kind, namespace, name, instance, reason, message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rec := eventRec{
		kind: kind, namespace: namespace, name: name, instance: instance,
		reason: reason, message: message, at: time.Now().UTC(),
	}
	if len(e.buf) >= e.cap {
		copy(e.buf, e.buf[1:])
		e.buf[len(e.buf)-1] = rec
		return
	}
	e.buf = append(e.buf, rec)
}

// list returns the events for a workload. instance=="" returns resource-level
// events; a non-empty instance returns that instance's events.
func (e *eventRing) list(kind, namespace, name, instance string) *eventsv1.EventList {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := &eventsv1.EventList{}
	for i := range e.buf {
		r := e.buf[i]
		if r.kind != kind || r.namespace != namespace || r.name != name || r.instance != instance {
			continue
		}
		out.Items = append(out.Items, toEvent(r))
	}
	return out
}

func toEvent(r eventRec) eventsv1.Event {
	regardingKind, regardingName := regardingFor(r)
	return eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%d", regardingName, r.at.UnixNano()),
			Namespace: r.namespace,
		},
		EventTime:           metav1.NewMicroTime(r.at),
		Reason:              r.reason,
		Note:                r.message,
		Type:                "Normal",
		ReportingController: "axisml-standalone-runtime",
		Regarding: corev1.ObjectReference{
			Kind:      regardingKind,
			Namespace: r.namespace,
			Name:      regardingName,
		},
	}
}

func regardingFor(r eventRec) (kind, name string) {
	if r.instance != "" {
		return "Pod", r.instance
	}
	switch r.kind {
	case KindRun:
		return "MLRun", r.name
	case KindService:
		return "MLService", r.name
	case KindTraffic:
		return "MLTrafficPolicy", r.name
	}
	return r.kind, r.name
}
