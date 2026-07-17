package trafficpolicy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	mltp "github.com/axisml/axisml/axisml-system/apis/mltrafficpolicy/v1alpha1"

	"github.com/axisml/axisml/axisml-system/compute-service/internal/server"
	"github.com/axisml/axisml/axisml-system/compute-service/internal/store"
	"github.com/axisml/axisml/axisml-system/compute-service/pkg/statusmap"
)

// reflectObserved reflects an observed MLTrafficPolicy CR status onto the row's
// PG (phase, status) via the shared statusmap (design §9.1). Shared by the
// Kubernetes informer and the standalone poller.
func reflectObserved(ctx context.Context, repo *Repository, row *store.TrafficPolicy, observed mltp.MLTrafficPolicyStatus) {
	if Status(row.Phase) == StatusDeleting || Status(row.Phase) == StatusDeleted {
		return
	}

	var sf server.TrafficPolicyStatus
	if len(row.StatusJSON) > 0 {
		_ = json.Unmarshal(row.StatusJSON, &sf)
	}
	cur := statusmap.TrafficStatus{Message: sf.Message, Endpoint: sf.Endpoint}
	for _, b := range sf.Backends {
		cur.Backends = append(cur.Backends, statusmap.TrafficBackend{
			ServiceName: b.ServiceName, Weight: b.Weight, Ready: b.Ready,
		})
	}
	newPhase, mapped := statusmap.MapTraffic(row.Phase, cur, observed)
	sf.Message = mapped.Message
	sf.Endpoint = mapped.Endpoint
	sf.Backends = sf.Backends[:0]
	for _, b := range mapped.Backends {
		sf.Backends = append(sf.Backends, server.TrafficPolicyBackendStatus{
			ServiceName: b.ServiceName,
			Weight:      b.Weight,
			Ready:       b.Ready,
		})
	}
	b, _ := json.Marshal(sf)

	_ = repo.Update(ctx, row.ID, map[string]any{
		"phase":  newPhase,
		"status": b,
	})
}

// reflectGone advances a row whose underlying route no longer exists: a
// Deleting row converges to Deleted; an active row that vanished externally is
// re-emitted (a traffic policy is pure declarative routing and PG is
// authoritative, so the reconciler rebuilds the CR — compute-service.md §5.5).
func reflectGone(ctx context.Context, repo *Repository, row *store.TrafficPolicy) {
	switch Status(row.Phase) {
	case StatusDeleting:
		_ = repo.Update(ctx, row.ID, map[string]any{
			"phase":      string(StatusDeleted),
			"deleted_at": time.Now().UTC(),
		})
	case StatusPending, StatusReady, StatusDegraded, StatusFailed:
		// Bump generation so the spec-sync predicate re-emits the CR.
		_ = repo.Update(ctx, row.ID, map[string]any{
			"observed_generation": row.Generation - 1,
		})
	}
}

func policyIDFromLabels(cr *mltp.MLTrafficPolicy) (uuid.UUID, error) {
	if v, ok := cr.Labels[mltp.LabelTrafficPolicyID]; ok {
		return uuid.Parse(v)
	}
	return uuid.Nil, errMissingID
}

var errMissingID = sentinel("missing compute.axisml.io/traffic-policy-id label")

type sentinel string

func (s sentinel) Error() string { return string(s) }
