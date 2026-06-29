package traffic

import (
	"encoding/json"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

type specShape struct {
	Mode     string `json:"mode"`
	Endpoint struct {
		Path     string `json:"path"`
		Hostname string `json:"hostname"`
	} `json:"endpoint"`
	Backends []struct {
		ServiceName string `json:"serviceName"`
		Role        string `json:"role"`
		Weight      int    `json:"weight"`
	} `json:"backends"`
}

type statusShape struct {
	Endpoint string `json:"endpoint"`
	Message  string `json:"message"`
	Backends []struct {
		ServiceName string `json:"serviceName"`
		Weight      int    `json:"weight"`
		Ready       bool   `json:"ready"`
	} `json:"backends"`
}

func decode(t *computeservice.TrafficPolicy) (specShape, statusShape) {
	var sp specShape
	if b, err := json.Marshal(t.Spec); err == nil {
		_ = json.Unmarshal(b, &sp)
	}
	var st statusShape
	if b, err := json.Marshal(t.Status); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return sp, st
}

func toView(t *computeservice.TrafficPolicy, tenant string) server.TrafficPolicy {
	sp, st := decode(t)
	v := server.TrafficPolicy{
		ID:          server.UUID(t.Id.String()),
		Namespace:   t.Namespace,
		TenantName:  tenant,
		Name:        t.Name,
		DisplayName: strv(t.DisplayName),
		Description: strv(t.Description),
		Owner:       strv(t.Owner),
		Mode:        server.TrafficPolicyMode(t.Mode),
		Endpoint:    server.TrafficPolicyEndpoint{Path: sp.Endpoint.Path, Hostname: sp.Endpoint.Hostname},
		AccessURL:   st.Endpoint,
		Phase:       server.TrafficPolicyPhase(t.Phase),
		Message:     st.Message,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	ready := map[string]bool{}
	for _, b := range st.Backends {
		ready[b.ServiceName] = b.Ready
	}
	total := 0
	for _, b := range sp.Backends {
		total += b.Weight
	}
	for _, b := range sp.Backends {
		pct := 0
		if total > 0 {
			pct = b.Weight * 100 / total
		}
		v.Backends = append(v.Backends, server.TrafficPolicyBackend{
			ServiceName: b.ServiceName,
			Role:        server.TrafficPolicyBackendRole(b.Role),
			Weight:      b.Weight,
			ActualPct:   pct,
			Ready:       ready[b.ServiceName],
		})
		if b.Role == "canary" {
			v.CanaryPercent = b.Weight
		}
	}
	return v
}

func strv(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
