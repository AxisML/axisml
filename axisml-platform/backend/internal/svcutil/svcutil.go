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
	ConfigMaps []server.WorkloadConfigMap `json:"configMaps"`
	Roles      []struct {
		Name     string `json:"name"`
		Replicas int    `json:"replicas"`
		Template struct {
			Image        string           `json:"image"`
			Command      []string         `json:"command"`
			Args         []string         `json:"args"`
			Env          []server.EnvVar  `json:"env"`
			EnvFrom      []map[string]any `json:"envFrom"`
			Volumes      []map[string]any `json:"volumes"`
			VolumeMounts []map[string]any `json:"volumeMounts"`
		} `json:"template"`
	} `json:"roles"`
	Route *struct {
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
	} `json:"route"`
}

type decodedStatus struct {
	AdmittedReplicas int    `json:"admittedReplicas"`
	ReadyReplicas    int    `json:"readyReplicas"`
	Endpoint         string `json:"endpoint"`
	Message          string `json:"message"`
	AdmissionReason  string `json:"admissionReason"`
	AdmissionMessage string `json:"admissionMessage"`
}

func decode(s *computeservice.MLService) (decodedSpec, decodedStatus) {
	var spec decodedSpec
	if b, err := json.Marshal(s.Spec); err == nil {
		_ = json.Unmarshal(b, &spec)
	}
	for i := range spec.Roles {
		tmpl := &spec.Roles[i].Template
		tmpl.EnvFrom = compactMaps(tmpl.EnvFrom)
		tmpl.Volumes = compactMaps(tmpl.Volumes)
		tmpl.VolumeMounts = compactMaps(tmpl.VolumeMounts)
	}
	var st decodedStatus
	if b, err := json.Marshal(s.Status); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return spec, st
}

// compactMaps removes nil fields introduced when the generated compute client
// marshals Kubernetes structs whose pointer fields lack omitempty tags. The
// Platform contract should return the same concise pass-through shape callers
// submitted, rather than every unused Kubernetes union member as null.
func compactMaps(in []map[string]any) []map[string]any {
	for i := range in {
		if compacted, ok := compactValue(in[i]); ok {
			in[i] = compacted.(map[string]any)
		}
	}
	return in
}

func compactValue(value any) (any, bool) {
	switch v := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			if compacted, ok := compactValue(item); ok {
				out[key] = compacted
			}
		}
		return out, true
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			if compacted, ok := compactValue(item); ok {
				out = append(out, compacted)
			}
		}
		return out, true
	default:
		return value, true
	}
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
	if len(req.EnvFrom) > 0 {
		tmpl["envFrom"] = req.EnvFrom
	}
	if len(req.Volumes) > 0 {
		tmpl["volumes"] = req.Volumes
	}
	if len(req.VolumeMounts) > 0 {
		tmpl["volumeMounts"] = req.VolumeMounts
	}
	if len(ports) > 0 {
		tmpl["ports"] = ports
	}
	input := map[string]any{
		"name":     req.Name,
		"kind":     "service",
		"poolName": req.PoolName,
		"unitName": req.UnitName,
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
	if len(req.ConfigMaps) > 0 {
		input["configMaps"] = req.ConfigMaps
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
	// Durable volumes are pre-provisioned by Platform via the DataVolumes catalog
	// (cluster-manager); the workspace only references an existing one by claim
	// name, written into the role template as a PVC volume + matching mount — the
	// same mechanism a Run/Service uses. Compute relays it and never provisions or
	// reclaims storage (backend.md §4.4).
	if len(req.Volumes) > 0 {
		volumes := make([]map[string]any, 0, len(req.Volumes))
		mounts := make([]map[string]any, 0, len(req.Volumes))
		for _, v := range req.Volumes {
			volumes = append(volumes, map[string]any{
				"name":                  v.Name,
				"persistentVolumeClaim": map[string]any{"claimName": v.Name},
			})
			mounts = append(mounts, map[string]any{"name": v.Name, "mountPath": v.MountPath})
		}
		tmpl["volumes"] = volumes
		tmpl["volumeMounts"] = mounts
	}
	input := map[string]any{
		"name":     req.Name,
		"kind":     "workspace",
		"poolName": req.PoolName,
		"unitName": req.UnitName,
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
		ID:               server.UUID(s.Id.String()),
		Namespace:        s.Namespace,
		TenantName:       tenant,
		Name:             s.Name,
		DisplayName:      strv(s.DisplayName),
		Description:      strv(s.Description),
		Owner:            strv(s.Owner),
		AdmittedReplicas: st.AdmittedReplicas,
		ReadyReplicas:    st.ReadyReplicas,
		Phase:            server.MLServicePhase(s.Phase),
		Message:          st.Message,
		AdmissionReason:  st.AdmissionReason,
		AdmissionMessage: st.AdmissionMessage,
		AccessURL:        st.Endpoint,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
		ConfigMaps:       spec.ConfigMaps,
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
		v.Env = spec.Roles[0].Template.Env
		v.EnvFrom = spec.Roles[0].Template.EnvFrom
		v.Volumes = spec.Roles[0].Template.Volumes
		v.VolumeMounts = spec.Roles[0].Template.VolumeMounts
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
		ID:               server.UUID(s.Id.String()),
		Namespace:        s.Namespace,
		TenantName:       tenant,
		Name:             s.Name,
		DisplayName:      strv(s.DisplayName),
		Description:      strv(s.Description),
		Owner:            strv(s.Owner),
		AdmittedReplicas: st.AdmittedReplicas,
		ReadyReplicas:    st.ReadyReplicas,
		Phase:            server.WorkspacePhase(s.Phase),
		Message:          st.Message,
		AdmissionReason:  st.AdmissionReason,
		AdmissionMessage: st.AdmissionMessage,
		Endpoint:         server.WorkspaceEndpoint{AccessURL: st.Endpoint},
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
	annos := derefMap(s.Annotations)
	v.PoolName = annos["resource.axisml.io/pool"]
	v.UnitName = annos["resource.axisml.io/unit"]
	if len(spec.Roles) > 0 {
		v.Replicas = spec.Roles[0].Replicas
		v.Image = spec.Roles[0].Template.Image
		v.Command = spec.Roles[0].Template.Command
		v.Args = spec.Roles[0].Template.Args
		for _, m := range spec.Roles[0].Template.VolumeMounts {
			name, _ := m["name"].(string)
			mountPath, _ := m["mountPath"].(string)
			v.Volumes = append(v.Volumes, server.WorkspaceVolume{Name: name, MountPath: mountPath})
		}
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
