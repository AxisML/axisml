package resourcepool

import (
	"encoding/json"
	"strings"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// ---- generated -> contract ----

func toPoolView(p *clustermanager.Pool) server.ResourcePool {
	v := server.ResourcePool{
		Name:            p.Name,
		Description:     deref(p.Description),
		NodeSelector:    server.StringMap(derefMap(p.NodeSelector)),
		Tolerations:     fromCMTolerations(p.Tolerations),
		Labels:          server.StringMap(derefMap(p.Labels)),
		Annotations:     server.StringMap(derefMap(p.Annotations)),
		ResourceVersion: deref(p.ResourceVersion),
		CreatedAt:       p.CreatedAt,
	}
	if p.UpdatedAt != nil {
		v.UpdatedAt = *p.UpdatedAt
	}
	for i := range p.Units {
		v.Units = append(v.Units, toUnitView(&p.Units[i]))
	}
	return v
}

func toUnitView(u *clustermanager.Unit) server.ResourceUnit {
	return server.ResourceUnit{
		Name:         u.Name,
		Description:  deref(u.Description),
		Requests:     server.ResourceMap(u.Requests),
		Limits:       server.ResourceMap(u.Limits),
		NodeSelector: server.StringMap(derefMap(u.NodeSelector)),
		Annotations:  server.StringMap(derefMap(u.Annotations)),
	}
}

// ---- contract -> generated request bodies ----

func poolCreateBody(req server.ResourcePoolCreateRequest) clustermanager.PoolCreate {
	body := clustermanager.PoolCreate{
		Name:         strPtrIf(req.Name),
		Description:  strPtrIf(req.Description),
		Labels:       mapPtrIf(req.Labels),
		Annotations:  mapPtrIf(req.Annotations),
		NodeSelector: mapPtrIf(req.NodeSelector),
		Tolerations:  toCMTolerations(req.Tolerations),
	}
	if len(req.Units) > 0 {
		units := make([]clustermanager.UnitInline, 0, len(req.Units))
		for _, u := range req.Units {
			units = append(units, clustermanager.UnitInline{
				Name:         u.Name,
				Description:  strPtrIf(u.Description),
				Requests:     map[string]string(u.Requests),
				Limits:       map[string]string(u.Limits),
				NodeSelector: mapPtrIf(u.NodeSelector),
				Annotations:  mapPtrIf(u.Annotations),
			})
		}
		body.Units = &units
	}
	return body
}

func poolPatchBody(req server.ResourcePoolPatchRequest) clustermanager.PoolPatch {
	return clustermanager.PoolPatch{
		Description:  strPtrIf(req.Description),
		Labels:       mapPtrIf(req.Labels),
		Annotations:  mapPtrIf(req.Annotations),
		NodeSelector: mapPtrIf(req.NodeSelector),
		Tolerations:  toCMTolerations(req.Tolerations),
	}
}

func unitCreateBody(req server.ResourceUnitCreateRequest) clustermanager.UnitCreate {
	return clustermanager.UnitCreate{
		Name:         strPtrIf(req.Name),
		Description:  strPtrIf(req.Description),
		Requests:     mapPtrIf(req.Requests),
		Limits:       mapPtrIf(req.Limits),
		NodeSelector: mapPtrIf(req.NodeSelector),
		Annotations:  mapPtrIf(req.Annotations),
	}
}

func unitPatchBody(req server.ResourceUnitPatchRequest) clustermanager.UnitPatch {
	return clustermanager.UnitPatch{
		Description:  strPtrIf(req.Description),
		Requests:     mapPtrIf(req.Requests),
		Limits:       mapPtrIf(req.Limits),
		NodeSelector: mapPtrIf(req.NodeSelector),
		Annotations:  mapPtrIf(req.Annotations),
	}
}

// ---- helpers ----

func matchPool(p server.ResourcePool, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(p.Name), q) ||
		strings.Contains(strings.ToLower(p.Description), q)
}

// fromCMTolerations / toCMTolerations bridge the contract's free-form Toleration
// (map) and cluster-manager's typed Corev1Toleration via JSON.
func fromCMTolerations(in *[]clustermanager.Toleration) []server.Toleration {
	if in == nil || len(*in) == 0 {
		return nil
	}
	out := make([]server.Toleration, 0, len(*in))
	for _, t := range *in {
		b, _ := json.Marshal(t)
		m := server.Toleration{}
		_ = json.Unmarshal(b, &m)
		out = append(out, m)
	}
	return out
}

func toCMTolerations(in []server.Toleration) *[]clustermanager.Toleration {
	if len(in) == 0 {
		return nil
	}
	out := make([]clustermanager.Toleration, 0, len(in))
	for _, m := range in {
		b, _ := json.Marshal(m)
		var t clustermanager.Toleration
		_ = json.Unmarshal(b, &t)
		out = append(out, t)
	}
	return &out
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefMap(p *map[string]string) map[string]string {
	if p == nil {
		return nil
	}
	return *p
}

func strPtrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mapPtrIf(m map[string]string) *map[string]string {
	if len(m) == 0 {
		return nil
	}
	return &m
}
