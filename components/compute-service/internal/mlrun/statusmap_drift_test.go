package mlrun

import (
	"testing"

	"github.com/axisml/axisml/components/compute-service/pkg/statusmap"
)

// TestPhaseValuesMatchStatusmap guards against drift between the internal Job
// phase enum and the published statusmap constants the informer reflows through.
func TestPhaseValuesMatchStatusmap(t *testing.T) {
	cases := map[Status]string{
		StatusCreating:  statusmap.RunCreating,
		StatusPending:   statusmap.RunPending,
		StatusRunning:   statusmap.RunRunning,
		StatusSucceeded: statusmap.RunSucceeded,
		StatusFailed:    statusmap.RunFailed,
		StatusCanceling: statusmap.RunCanceling,
		StatusCancelled: statusmap.RunCancelled,
	}
	for internal, shared := range cases {
		if string(internal) != shared {
			t.Errorf("phase drift: internal %q != statusmap %q", internal, shared)
		}
	}
}
