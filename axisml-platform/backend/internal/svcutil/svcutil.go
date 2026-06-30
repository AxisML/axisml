// Package svcutil holds the shared MLService logic for the MLServices and
// Workspaces modules: building a compute MLService create request and projecting
// the compute MLService view into the contract MLService / Workspace types.
package svcutil

import (
	"encoding/json"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// LastReplicasAnnotation records the replica count before a stop, so start can
// restore it (§5.5).
const LastReplicasAnnotation = "platform.axisml.io/last-replicas"

// decodedSpec is the subset of the compute MLService spec we project from.
type decodedSpec struct {
	Roles []struct {
		Name     string `json:"name"`
		Replicas int    `json:"replicas"`
		Template struct {
			Image   string   `json:"image"`
			Command []string `json:"command"`
			Args    []string `json:"args"`
		} `json:"template"`
	} `json:"roles"`
	Scheduling struct {
		Quota string `json:"quota"`
	} `json:"scheduling"`
	Route *struct {
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
	} `json:"route"`
}

type decodedStatus struct {
	ReadyReplicas int    `json:"readyReplicas"`
	Endpoint      string `json:"endpoint"`
	Message       string `json:"message"`
}

func decode(s *computeservice.MLService) (decodedSpec, decodedStatus) {
	var spec decodedSpec
	if b, err := json.Marshal(s.Spec); err == nil {
		_ = json.Unmarshal(b, &spec)
	}
	var st decodedStatus
	if b, err := json.Marshal(s.Status); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return spec, st
}

// BuildServiceInput assembles a kind=service MLService create request from the
// contract request (a single "default" role carries the image/ports/command).
func BuildServiceInput(req server.MLServiceCreateRequest) (computeservice.MLServiceCreate, error) {
	ports := make([]map[string]any, 0, len(req.Ports))
	for _, p := range req.Ports {
		ports = append(ports, map[string]any{"name": p.Name, "containerPort": p.Port})
	}
	tmpl := map[string]any{"image": req.Image}
	if len(req.Command) > 0 {
		tmpl["command"] = req.Command
	}
	if len(req.Args) > 0 {
		tmpl["args"] = req.Args
	}
	if len(req.Env) > 0 {
		tmpl["env"] = req.Env
	}
	if len(ports) > 0 {
		tmpl["ports"] = ports
	}
	input := map[string]any{
		"name":     req.Name,
		"kind":     "service",
		"poolName": req.PoolName,
		"unitName": req.UnitName,
		"quota":    req.Quota,
		"roles":    []map[string]any{{"name": "default", "replicas": req.Replicas, "template": tmpl}},
		"route":    map[string]any{"enabled": req.Route.Enabled, "path": req.Route.Path},
		// Surface the model reference + pool/unit for display (compute expands
		// pool/unit away in the stored spec).
		"annotations": map[string]any{
			"platform.axisml.io/model-name":    req.ModelName,
			"platform.axisml.io/model-version": req.ModelVersion,
			"resource.axisml.io/pool":          req.PoolName,
			"resource.axisml.io/unit":          req.UnitName,
		},
	}
	if req.DisplayName != "" {
		input["displayName"] = req.DisplayName
	}
	if req.Description != "" {
		input["description"] = req.Description
	}
	if req.Backend.Name != "" || req.Backend.Engine != "" {
		input["backend"] = map[string]any{"name": req.Backend.Name, "engine": req.Backend.Engine, "config": req.Backend.Config}
	}
	return marshalInput(input)
}

// BuildWorkspaceInput assembles a kind=workspace MLService create request.
func BuildWorkspaceInput(req server.WorkspaceCreateRequest) (computeservice.MLServiceCreate, error) {
	tmpl := map[string]any{"image": req.Image}
	if len(req.Command) > 0 {
		tmpl["command"] = req.Command
	}
	if len(req.Args) > 0 {
		tmpl["args"] = req.Args
	}
	if len(req.Env) > 0 {
		tmpl["env"] = req.Env
	}
	if req.ContainerPort > 0 {
		tmpl["ports"] = []map[string]any{{"name": "http", "containerPort": req.ContainerPort}}
	}
	input := map[string]any{
		"name":     req.Name,
		"kind":     "workspace",
		"poolName": req.PoolName,
		"unitName": req.UnitName,
		"quota":    req.Quota,
		"roles":    []map[string]any{{"name": "workspace", "replicas": 1, "template": tmpl}},
		"route":    map[string]any{"enabled": false},
		"annotations": map[string]any{
			"resource.axisml.io/pool": req.PoolName,
			"resource.axisml.io/unit": req.UnitName,
		},
	}
	if req.DisplayName != "" {
		input["displayName"] = req.DisplayName
	}
	if req.Description != "" {
		input["description"] = req.Description
	}
	// Durable volumes are pre-provisioned by Platform via cluster-manager and
	// injected into the role template as PVC volume/volumeMount entries. That
	// orchestration is wired up separately; compute no longer provisions storage.
	return marshalInput(input)
}

func marshalInput(m map[string]any) (computeservice.MLServiceCreate, error) {
	var out computeservice.MLServiceCreate
	b, err := json.Marshal(m)
	if err != nil {
		return out, err
	}
	return out, json.Unmarshal(b, &out)
}

// ServiceToView projects a compute MLService view into the contract MLService.
func ServiceToView(s *computeservice.MLService, tenant string) server.MLService {
	spec, st := decode(s)
	v := server.MLService{
		ID:            server.UUID(s.Id.String()),
		Namespace:     s.Namespace,
		TenantName:    tenant,
		Name:          s.Name,
		DisplayName:   strv(s.DisplayName),
		Description:   strv(s.Description),
		Owner:         strv(s.Owner),
		Quota:         spec.Scheduling.Quota,
		ReadyReplicas: st.ReadyReplicas,
		Phase:         server.MLServicePhase(s.Phase),
		Message:       st.Message,
		AccessURL:     st.Endpoint,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	annos := derefMap(s.Annotations)
	v.ModelName = annos["platform.axisml.io/model-name"]
	v.ModelVersion = annos["platform.axisml.io/model-version"]
	v.PoolName = annos["resource.axisml.io/pool"]
	v.UnitName = annos["resource.axisml.io/unit"]
	if len(spec.Roles) > 0 {
		v.Replicas = spec.Roles[0].Replicas
		v.Image = spec.Roles[0].Template.Image
		v.Command = spec.Roles[0].Template.Command
		v.Args = spec.Roles[0].Template.Args
	}
	if spec.Route != nil {
		v.Route = server.MLServiceRoute{Enabled: spec.Route.Enabled, Path: spec.Route.Path}
	}
	return v
}

// WorkspaceToView projects a compute workspace MLService view into the contract
// Workspace.
func WorkspaceToView(s *computeservice.MLService, tenant string) server.Workspace {
	spec, st := decode(s)
	v := server.Workspace{
		ID:            server.UUID(s.Id.String()),
		Namespace:     s.Namespace,
		TenantName:    tenant,
		Name:          s.Name,
		DisplayName:   strv(s.DisplayName),
		Description:   strv(s.Description),
		Owner:         strv(s.Owner),
		Quota:         spec.Scheduling.Quota,
		ReadyReplicas: st.ReadyReplicas,
		Phase:         server.WorkspacePhase(s.Phase),
		Message:       st.Message,
		Endpoint:      server.WorkspaceEndpoint{AccessURL: st.Endpoint},
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	annos := derefMap(s.Annotations)
	v.PoolName = annos["resource.axisml.io/pool"]
	v.UnitName = annos["resource.axisml.io/unit"]
	if len(spec.Roles) > 0 {
		v.Replicas = spec.Roles[0].Replicas
		v.Image = spec.Roles[0].Template.Image
		v.Command = spec.Roles[0].Template.Command
		v.Args = spec.Roles[0].Template.Args
	}
	if v.Replicas > 0 {
		v.DesiredState = "Running"
	} else {
		v.DesiredState = "Stopped"
	}
	return v
}

func strv(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefMap(p *map[string]string) map[string]string {
	if p == nil {
		return map[string]string{}
	}
	return *p
}
