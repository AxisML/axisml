package trafficpolicy

import (
	"testing"

	"github.com/axisml/axisml/axisml-system/compute-service/pkg/statusmap"
)

// TestPhaseValuesMatchStatusmap guards against drift between the internal
// TrafficPolicy phase enum and the published statusmap constants.
func TestPhaseValuesMatchStatusmap(t *testing.T) {
	cases := map[Status]string{
		StatusPending:  statusmap.ServicePending,
		StatusReady:    statusmap.ServiceReady,
		StatusDegraded: statusmap.ServiceDegraded,
		StatusFailed:   statusmap.ServiceFailed,
	}
	for internal, shared := range cases {
		if string(internal) != shared {
			t.Errorf("phase drift: internal %q != statusmap %q", internal, shared)
		}
	}
}
