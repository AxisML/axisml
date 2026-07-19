package runutil_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/runutil"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
)

func baseSpec() server.JobSpec {
	return server.JobSpec{
		Backend:  server.Backend{Name: "native", Engine: "job"},
		PoolName: "pool-a", UnitName: "unit-a",
		Roles: []server.MLRunRole{{
			Name:          "master",
			Replicas:      2,
			RestartPolicy: server.RestartPolicy("Never"),
			Template: server.RoleTemplate{
				Image:     "img:1",
				Command:   []string{"python"},
				Args:      []string{"train.py"},
				Env:       []server.EnvVar{{Name: "K", Value: "V"}},
				Resources: server.ResourceMap{"cpu": "2"},
			},
		}},
		RunPolicy: server.RunPolicy{ActiveDeadlineSeconds: 100, BackoffLimit: 3, TTLSecondsAfterFinished: 60},
	}
}

func TestBuildRunInput_Base(t *testing.T) {
	labels := map[string]string{"team": "ml"}
	annos := map[string]string{"note": "x"}
	out, err := runutil.BuildRunInput(baseSpec(), nil, "job-1", "My Run", labels, annos)
	require.NoError(t, err)

	assert.Equal(t, "job-1", out.Name)
	assert.Equal(t, "pool-a", out.PoolName)
	assert.Equal(t, "unit-a", out.UnitName)

	require.NotNil(t, out.DisplayName)
	assert.Equal(t, "My Run", *out.DisplayName)
	require.NotNil(t, out.Labels)
	assert.Equal(t, "ml", (*out.Labels)["team"])
	require.NotNil(t, out.Annotations)
	assert.Equal(t, "x", (*out.Annotations)["note"])

	require.NotNil(t, out.Backend)
	assert.Equal(t, "native", out.Backend.Name)
	assert.Equal(t, "job", out.Backend.Engine)

	require.Len(t, out.Roles, 1)
	r := out.Roles[0]
	assert.Equal(t, "master", r.Name)
	assert.Equal(t, int32(2), r.Replicas)
	require.NotNil(t, r.RestartPolicy)
	assert.Equal(t, "Never", *r.RestartPolicy)
	assert.Equal(t, "img:1", r.Template.Image)
	require.NotNil(t, r.Template.Command)
	assert.Equal(t, []string{"python"}, *r.Template.Command)
	require.NotNil(t, r.Template.Args)
	assert.Equal(t, []string{"train.py"}, *r.Template.Args)
	require.NotNil(t, r.Template.Env)
	require.Len(t, *r.Template.Env, 1)
	assert.Equal(t, "K", (*r.Template.Env)[0].Name)
	require.NotNil(t, (*r.Template.Env)[0].Value)
	assert.Equal(t, "V", *(*r.Template.Env)[0].Value)
	// Resources are stamped onto both requests and limits.
	require.NotNil(t, r.Template.Resources)
	require.NotNil(t, r.Template.Resources.Requests)
	assert.Equal(t, "2", (*r.Template.Resources.Requests)["cpu"])
	require.NotNil(t, r.Template.Resources.Limits)
	assert.Equal(t, "2", (*r.Template.Resources.Limits)["cpu"])

	require.NotNil(t, out.RunPolicy)
	require.NotNil(t, out.RunPolicy.ActiveDeadlineSeconds)
	assert.Equal(t, int64(100), *out.RunPolicy.ActiveDeadlineSeconds)
	require.NotNil(t, out.RunPolicy.BackoffLimit)
	assert.Equal(t, int32(3), *out.RunPolicy.BackoffLimit)
	require.NotNil(t, out.RunPolicy.TtlSecondsAfterFinished)
	assert.Equal(t, int32(60), *out.RunPolicy.TtlSecondsAfterFinished)
}

func TestBuildRunInput_ForwardsVolumes(t *testing.T) {
	// A PVC-backed dataset volume declared on the role template must reach the
	// compute MLRun create request intact — name, mount, and the volume source.
	spec := baseSpec()
	spec.Roles[0].Template.Volumes = []map[string]any{
		{"name": "data", "persistentVolumeClaim": map[string]any{"claimName": "dataset-1"}},
	}
	spec.Roles[0].Template.VolumeMounts = []map[string]any{
		{"name": "data", "mountPath": "/data"},
	}
	out, err := runutil.BuildRunInput(spec, nil, "job-1", "", nil, nil)
	require.NoError(t, err)
	require.Len(t, out.Roles, 1)
	tmpl := out.Roles[0].Template

	require.NotNil(t, tmpl.Volumes)
	require.Len(t, *tmpl.Volumes, 1)
	vol := (*tmpl.Volumes)[0]
	assert.Equal(t, "data", vol.Name)
	require.NotNil(t, vol.PersistentVolumeClaim, "volume source must survive the typed round-trip")
	assert.Equal(t, "dataset-1", vol.PersistentVolumeClaim.ClaimName)

	require.NotNil(t, tmpl.VolumeMounts)
	require.Len(t, *tmpl.VolumeMounts, 1)
	assert.Equal(t, "data", (*tmpl.VolumeMounts)[0].Name)
	assert.Equal(t, "/data", (*tmpl.VolumeMounts)[0].MountPath)
}

func TestBuildRunInput_ForwardsConfigMapSources(t *testing.T) {
	spec := baseSpec()
	spec.ConfigMaps = []server.WorkloadConfigMap{{
		Name: "run-config",
		Data: map[string]string{"log-level": "debug"},
	}}
	spec.Roles[0].Template.Env = []server.EnvVar{{
		Name: "LOG_LEVEL",
		ValueFrom: map[string]any{
			"configMapKeyRef": map[string]any{"name": "run-config", "key": "log-level"},
		},
	}}
	spec.Roles[0].Template.EnvFrom = []map[string]any{{
		"prefix":       "APP_",
		"configMapRef": map[string]any{"name": "run-config"},
	}}
	spec.Roles[0].Template.Volumes = []map[string]any{{
		"name":      "config",
		"configMap": map[string]any{"name": "run-config"},
	}}
	spec.Roles[0].Template.VolumeMounts = []map[string]any{{
		"name": "config", "mountPath": "/etc/run", "readOnly": true,
	}}

	out, err := runutil.BuildRunInput(spec, nil, "job-config", "", nil, nil)
	require.NoError(t, err)
	tmpl := out.Roles[0].Template

	require.NotNil(t, tmpl.Env)
	require.Len(t, *tmpl.Env, 1)
	require.NotNil(t, (*tmpl.Env)[0].ValueFrom)
	require.NotNil(t, (*tmpl.Env)[0].ValueFrom.ConfigMapKeyRef)
	assert.Equal(t, "run-config", *(*tmpl.Env)[0].ValueFrom.ConfigMapKeyRef.Name)
	assert.Equal(t, "log-level", (*tmpl.Env)[0].ValueFrom.ConfigMapKeyRef.Key)

	require.NotNil(t, tmpl.EnvFrom)
	require.Len(t, *tmpl.EnvFrom, 1)
	require.NotNil(t, (*tmpl.EnvFrom)[0].ConfigMapRef)
	assert.Equal(t, "run-config", *(*tmpl.EnvFrom)[0].ConfigMapRef.Name)
	assert.Equal(t, "APP_", *(*tmpl.EnvFrom)[0].Prefix)

	require.NotNil(t, tmpl.Volumes)
	require.Len(t, *tmpl.Volumes, 1)
	require.NotNil(t, (*tmpl.Volumes)[0].ConfigMap)
	assert.Equal(t, "run-config", *(*tmpl.Volumes)[0].ConfigMap.Name)
	require.NotNil(t, tmpl.VolumeMounts)
	require.Len(t, *tmpl.VolumeMounts, 1)
	assert.Equal(t, "/etc/run", (*tmpl.VolumeMounts)[0].MountPath)
	require.NotNil(t, (*tmpl.VolumeMounts)[0].ReadOnly)
	assert.True(t, *(*tmpl.VolumeMounts)[0].ReadOnly)

	wire, err := json.Marshal(out)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(wire, &body))
	configMaps, _ := body["configMaps"].([]any)
	require.Len(t, configMaps, 1)
	assert.Equal(t, "run-config", configMaps[0].(map[string]any)["name"])
	assert.Equal(t, "debug", configMaps[0].(map[string]any)["data"].(map[string]any)["log-level"])
}

func TestBuildRunInput_OverridePrecedence(t *testing.T) {
	// Trigger overrides win over the spec for pool/unit, resources (all
	// roles), and per-role args/env.
	ovJSON := `{
		"poolName":"pool-b","unitName":"unit-b",
		"resources":{"cpu":"8"},
		"roles":[{"name":"master","args":["eval.py"],"env":[{"name":"O","value":"1"}]}]
	}`
	var ov server.RunTriggerRequest
	require.NoError(t, json.Unmarshal([]byte(ovJSON), &ov))

	out, err := runutil.BuildRunInput(baseSpec(), &ov, "job-2", "", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "pool-b", out.PoolName)
	assert.Equal(t, "unit-b", out.UnitName)

	require.Len(t, out.Roles, 1)
	r := out.Roles[0]
	require.NotNil(t, r.Template.Args)
	assert.Equal(t, []string{"eval.py"}, *r.Template.Args)
	require.NotNil(t, r.Template.Env)
	require.Len(t, *r.Template.Env, 1)
	assert.Equal(t, "O", (*r.Template.Env)[0].Name)
	// Resource override replaces the spec's per-role resources.
	require.NotNil(t, r.Template.Resources)
	require.NotNil(t, r.Template.Resources.Requests)
	assert.Equal(t, "8", (*r.Template.Resources.Requests)["cpu"])

	// Empty displayName / labels / annotations are omitted.
	assert.Nil(t, out.DisplayName)
	assert.Nil(t, out.Labels)
	assert.Nil(t, out.Annotations)
}

func TestBuildRunInput_EmptyOverrideKeepsSpec(t *testing.T) {
	// An override with empty scalar fields must not clobber the spec values.
	ov := &server.RunTriggerRequest{}
	out, err := runutil.BuildRunInput(baseSpec(), ov, "job-3", "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "pool-a", out.PoolName)
	assert.Equal(t, "unit-a", out.UnitName)
}

func TestBuildRunInput_EmptyTemplateOmitsFields(t *testing.T) {
	spec := server.JobSpec{
		// No backend, no pool/unit, no run policy.
		Roles: []server.MLRunRole{{
			Name:     "worker",
			Template: server.RoleTemplate{Image: "img:only"},
		}},
	}
	out, err := runutil.BuildRunInput(spec, nil, "job-4", "", nil, nil)
	require.NoError(t, err)

	require.Len(t, out.Roles, 1)
	r := out.Roles[0]
	assert.Equal(t, "img:only", r.Template.Image)
	assert.Nil(t, r.Template.Command)
	assert.Nil(t, r.Template.Args)
	assert.Nil(t, r.Template.Env)
	assert.Nil(t, r.Template.EnvFrom)
	assert.Nil(t, r.Template.Resources)
	assert.Nil(t, r.Template.Volumes)
	assert.Nil(t, r.Template.VolumeMounts)
	assert.Equal(t, int32(0), r.Replicas)
	assert.Nil(t, r.RestartPolicy)

	assert.Nil(t, out.Backend)
	assert.Nil(t, out.RunPolicy)
	assert.Nil(t, out.DisplayName)
	assert.Nil(t, out.Labels)
	assert.Nil(t, out.Annotations)
}

func TestBuildRunInput_BackendConfigIncluded(t *testing.T) {
	spec := baseSpec()
	spec.Backend = server.Backend{Name: "custom", Engine: "", Config: map[string]any{"gvk": "batch/v1/Job"}}
	out, err := runutil.BuildRunInput(spec, nil, "job-5", "", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, out.Backend)
	assert.Equal(t, "custom", out.Backend.Name)
	require.NotNil(t, out.Backend.Config)
	assert.Equal(t, "batch/v1/Job", (*out.Backend.Config)["gvk"])
}

func TestBuildRunInput_RunPolicyMap(t *testing.T) {
	tests := []struct {
		name    string
		policy  server.RunPolicy
		wantNil bool
		wantADS *int64
		wantBL  *int32
		wantTTL *int32
	}{
		{name: "empty", policy: server.RunPolicy{}, wantNil: true},
		{
			name:    "deadline only",
			policy:  server.RunPolicy{ActiveDeadlineSeconds: 5},
			wantADS: ptrInt64(5),
		},
		{
			name:    "all positive",
			policy:  server.RunPolicy{ActiveDeadlineSeconds: 5, BackoffLimit: 2, TTLSecondsAfterFinished: 9},
			wantADS: ptrInt64(5), wantBL: ptrInt32(2), wantTTL: ptrInt32(9),
		},
		{
			name:    "progress deadline ignored",
			policy:  server.RunPolicy{ProgressDeadlineSeconds: 30},
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := baseSpec()
			spec.RunPolicy = tt.policy
			out, err := runutil.BuildRunInput(spec, nil, "job", "", nil, nil)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, out.RunPolicy)
				return
			}
			require.NotNil(t, out.RunPolicy)
			assert.Equal(t, tt.wantADS, out.RunPolicy.ActiveDeadlineSeconds)
			assert.Equal(t, tt.wantBL, out.RunPolicy.BackoffLimit)
			assert.Equal(t, tt.wantTTL, out.RunPolicy.TtlSecondsAfterFinished)
		})
	}
}

func TestRunToView_Full(t *testing.T) {
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	dn, desc, owner, msg := "Disp", "A desc", "alice", "running now"
	started := time.Unix(1700000000, 0).UTC()
	finished := time.Unix(1700003600, 0).UTC()
	created := time.Unix(1699990000, 0).UTC()
	updated := time.Unix(1699995000, 0).UTC()

	r := &computeservice.MLRun{
		Id:          id,
		Namespace:   "ns-acme",
		Name:        "job-1-1",
		Phase:       "Running",
		DisplayName: &dn,
		Description: &desc,
		Owner:       &owner,
		CreatedAt:   created,
		UpdatedAt:   updated,
		Spec: gen.MLRunSpec{
			Scheduling: gen.MLRunSchedulingSpec{Quota: "quota-a"},
			ConfigMaps: &[]gen.WorkloadconfigConfigMap{{
				Name: "run-config",
				Data: &map[string]string{"trainer.yaml": "epochs: 3"},
			}},
		},
		Status: gen.MLRunStatus{Message: &msg, StartedAt: &started, FinishedAt: &finished},
	}
	v := runutil.RunToView(r, "acme", "job-1")

	assert.Equal(t, server.UUID(id.String()), v.ID)
	assert.Equal(t, "ns-acme", v.Namespace)
	assert.Equal(t, "acme", v.TenantName)
	assert.Equal(t, "job-1-1", v.Name)
	assert.Equal(t, "job-1", v.JobName)
	assert.Equal(t, "Disp", v.DisplayName)
	assert.Equal(t, "A desc", v.Description)
	assert.Equal(t, "alice", v.Owner)
	assert.Equal(t, server.RunPhase("Running"), v.Phase)
	assert.Equal(t, created, v.CreatedAt)
	assert.Equal(t, updated, v.UpdatedAt)

	assert.Empty(t, v.PoolName)
	assert.Empty(t, v.UnitName)
	require.Len(t, v.Spec.ConfigMaps, 1)
	assert.Equal(t, "run-config", v.Spec.ConfigMaps[0].Name)
	assert.Equal(t, "epochs: 3", v.Spec.ConfigMaps[0].Data["trainer.yaml"])

	// Status passthrough.
	assert.Equal(t, "running now", v.Message)
	require.NotNil(t, v.StartedAt)
	assert.Equal(t, started, *v.StartedAt)
	require.NotNil(t, v.FinishedAt)
	assert.Equal(t, finished, *v.FinishedAt)
}

func TestRunToView_NilOptionals(t *testing.T) {
	r := &computeservice.MLRun{
		Id:        uuid.Nil,
		Namespace: "ns",
		Name:      "run",
		Phase:     "Pending",
	}
	v := runutil.RunToView(r, "acme", "job")
	assert.Empty(t, v.DisplayName)
	assert.Empty(t, v.Description)
	assert.Empty(t, v.Owner)
	assert.Empty(t, v.Message)
	assert.Nil(t, v.StartedAt)
	assert.Nil(t, v.FinishedAt)
}

func ptrInt64(v int64) *int64 { return &v }
func ptrInt32(v int32) *int32 { return &v }
