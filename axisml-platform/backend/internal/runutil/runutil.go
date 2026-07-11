// Package runutil holds the shared Run logic used by both the Jobs and
// Experiments modules: building a compute MLRun create request from a definition
// spec ⊕ trigger overrides, and projecting a compute MLRun view into the contract
// Run. The two modules differ only in the grouping label key and naming.
package runutil

import (
	"encoding/json"
	"time"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

// BuildRunInput snapshots a definition spec with the trigger-time override
// whitelist (§4.2) into a compute MLRun create request. It builds the wire shape
// as JSON so the deeply-nested generated types are populated by tag, not by hand.
func BuildRunInput(spec server.JobSpec, ov *server.RunTriggerRequest, runName, displayName string, labels, annotations map[string]string) (computeservice.MLRunCreate, error) {
	pool, unit, quota := spec.PoolName, spec.UnitName, spec.Quota
	var resourceOverride server.ResourceMap
	roleArgs := map[string][]string{}
	roleEnv := map[string][]server.EnvVar{}
	if ov != nil {
		if ov.PoolName != "" {
			pool = ov.PoolName
		}
		if ov.UnitName != "" {
			unit = ov.UnitName
		}
		if ov.Quota != "" {
			quota = ov.Quota
		}
		resourceOverride = ov.Resources
		for _, r := range ov.Roles {
			if r.Args != nil {
				roleArgs[r.Name] = r.Args
			}
			if r.Env != nil {
				roleEnv[r.Name] = r.Env
			}
		}
	}

	roles := make([]map[string]any, 0, len(spec.Roles))
	for _, role := range spec.Roles {
		t := role.Template
		args := t.Args
		if v, ok := roleArgs[role.Name]; ok {
			args = v
		}
		env := t.Env
		if v, ok := roleEnv[role.Name]; ok {
			env = v
		}
		resources := t.Resources
		if resourceOverride != nil {
			resources = resourceOverride
		}
		tmpl := map[string]any{"image": t.Image}
		if len(t.Command) > 0 {
			tmpl["command"] = t.Command
		}
		if len(args) > 0 {
			tmpl["args"] = args
		}
		if len(env) > 0 {
			tmpl["env"] = env
		}
		if len(resources) > 0 {
			tmpl["resources"] = map[string]any{"requests": resources, "limits": resources}
		}
		// volumes / volumeMounts are pass-through K8s shapes: forward them so a
		// PVC-backed dataset volume the caller declares reaches the MLRun CR (and
		// thus the operator's pod template). A mount referencing an undeclared
		// volume is rejected downstream by the native-job handler's Validate.
		if len(t.Volumes) > 0 {
			tmpl["volumes"] = t.Volumes
		}
		if len(t.VolumeMounts) > 0 {
			tmpl["volumeMounts"] = t.VolumeMounts
		}
		rm := map[string]any{"name": role.Name, "template": tmpl}
		if role.Replicas > 0 {
			rm["replicas"] = role.Replicas
		}
		if role.RestartPolicy != "" {
			rm["restartPolicy"] = role.RestartPolicy
		}
		roles = append(roles, rm)
	}

	input := map[string]any{
		"name":     runName,
		"poolName": pool,
		"unitName": unit,
		"quota":    quota,
		"roles":    roles,
	}
	if displayName != "" {
		input["displayName"] = displayName
	}
	if len(labels) > 0 {
		input["labels"] = labels
	}
	if len(annotations) > 0 {
		input["annotations"] = annotations
	}
	if spec.Backend.Name != "" || spec.Backend.Engine != "" {
		b := map[string]any{"name": spec.Backend.Name, "engine": spec.Backend.Engine}
		if spec.Backend.Config != nil {
			b["config"] = spec.Backend.Config
		}
		input["backend"] = b
	}
	if rp := runPolicyMap(spec.RunPolicy); rp != nil {
		input["runPolicy"] = rp
	}

	var out computeservice.MLRunCreate
	b, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func runPolicyMap(p server.RunPolicy) map[string]any {
	m := map[string]any{}
	if p.ActiveDeadlineSeconds > 0 {
		m["activeDeadlineSeconds"] = p.ActiveDeadlineSeconds
	}
	if p.BackoffLimit > 0 {
		m["backoffLimit"] = p.BackoffLimit
	}
	if p.TTLSecondsAfterFinished > 0 {
		m["ttlSecondsAfterFinished"] = p.TTLSecondsAfterFinished
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// RunToView projects a compute MLRun view into the contract Run. tenantName
// is the active tenant; defName is the owning Job/Experiment name.
func RunToView(r *computeservice.MLRun, tenantName, defName string) server.Run {
	v := server.Run{
		ID:          server.UUID(r.Id.String()),
		Namespace:   r.Namespace,
		TenantName:  tenantName,
		Name:        r.Name,
		JobName:     defName,
		DisplayName: strv(r.DisplayName),
		Description: strv(r.Description),
		Owner:       strv(r.Owner),
		Phase:       server.RunPhase(r.Phase),
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	// Pull scheduling convenience fields by re-reading the spec as JSON.
	var spec struct {
		Scheduling map[string]any `json:"scheduling"`
	}
	if b, err := json.Marshal(r.Spec); err == nil {
		_ = json.Unmarshal(b, &spec)
	}
	if spec.Scheduling != nil {
		v.PoolName, _ = spec.Scheduling["poolName"].(string)
		v.UnitName, _ = spec.Scheduling["unitName"].(string)
		v.Quota, _ = spec.Scheduling["quota"].(string)
	}
	// Status passthrough (message/started/finished).
	var st struct {
		Message    string     `json:"message"`
		StartedAt  *time.Time `json:"startedAt"`
		FinishedAt *time.Time `json:"finishedAt"`
	}
	if b, err := json.Marshal(r.Status); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	v.Message = st.Message
	v.StartedAt = st.StartedAt
	v.FinishedAt = st.FinishedAt
	return v
}

func strv(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
