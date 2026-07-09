package svcutil_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/computeservice/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/server"
	"github.com/axisml/axisml/axisml-platform/backend/internal/svcutil"
)

func sp(s string) *string { return &s }

func TestBuildServiceInput(t *testing.T) {
	req := server.MLServiceCreateRequest{
		Name:         "svc-1",
		DisplayName:  "My Svc",
		Description:  "d",
		Backend:      server.Backend{Name: "kserve", Engine: "inference"},
		ModelName:    "resnet",
		ModelVersion: "v2",
		Image:        "img:svc",
		Command:      []string{"serve"},
		Args:         []string{"--port=8080"},
		Env:          []server.EnvVar{{Name: "E", Value: "1"}},
		Ports:        []server.ServicePort{{Name: "http", Port: 8080}},
		PoolName:     "pool-a",
		UnitName:     "unit-a",
		Quota:        "quota-a",
		Replicas:     3,
		Route:        server.MLServiceRoute{Enabled: true, Path: "/svc"},
	}
	out, err := svcutil.BuildServiceInput(req)
	require.NoError(t, err)

	assert.Equal(t, "svc-1", out.Name)
	require.NotNil(t, out.Kind)
	assert.Equal(t, "service", *out.Kind)
	assert.Equal(t, "pool-a", out.PoolName)
	assert.Equal(t, "unit-a", out.UnitName)
	assert.Equal(t, "quota-a", out.Quota)

	require.NotNil(t, out.DisplayName)
	assert.Equal(t, "My Svc", *out.DisplayName)
	require.NotNil(t, out.Description)
	assert.Equal(t, "d", *out.Description)

	require.NotNil(t, out.Backend)
	assert.Equal(t, "kserve", out.Backend.Name)
	assert.Equal(t, "inference", out.Backend.Engine)

	require.Len(t, out.Roles, 1)
	role := out.Roles[0]
	assert.Equal(t, "default", role.Name)
	assert.Equal(t, int32(3), role.Replicas)
	assert.Equal(t, "img:svc", role.Template.Image)
	require.NotNil(t, role.Template.Command)
	assert.Equal(t, []string{"serve"}, *role.Template.Command)
	require.NotNil(t, role.Template.Args)
	assert.Equal(t, []string{"--port=8080"}, *role.Template.Args)
	require.NotNil(t, role.Template.Env)
	require.Len(t, *role.Template.Env, 1)
	assert.Equal(t, "E", (*role.Template.Env)[0].Name)
	require.NotNil(t, role.Template.Ports)
	require.Len(t, *role.Template.Ports, 1)
	assert.Equal(t, "http", (*role.Template.Ports)[0].Name)
	assert.Equal(t, int32(8080), (*role.Template.Ports)[0].ContainerPort)

	require.NotNil(t, out.Route)
	assert.True(t, out.Route.Enabled)
	require.NotNil(t, out.Route.Path)
	assert.Equal(t, "/svc", *out.Route.Path)

	require.NotNil(t, out.Annotations)
	annos := *out.Annotations
	assert.Equal(t, "resnet", annos["platform.axisml.io/model-name"])
	assert.Equal(t, "v2", annos["platform.axisml.io/model-version"])
	assert.Equal(t, "pool-a", annos["resource.axisml.io/pool"])
	assert.Equal(t, "unit-a", annos["resource.axisml.io/unit"])
}

func TestBuildServiceInput_MinimalOmitsOptionals(t *testing.T) {
	req := server.MLServiceCreateRequest{
		Name:         "svc-2",
		ModelName:    "m",
		ModelVersion: "1",
		Image:        "img",
		Ports:        []server.ServicePort{{Name: "http", Port: 80}},
		PoolName:     "p",
		UnitName:     "u",
		Replicas:     1,
	}
	out, err := svcutil.BuildServiceInput(req)
	require.NoError(t, err)
	require.Len(t, out.Roles, 1)
	assert.Nil(t, out.Roles[0].Template.Command)
	assert.Nil(t, out.Roles[0].Template.Args)
	assert.Nil(t, out.Roles[0].Template.Env)
	assert.Nil(t, out.DisplayName)
	assert.Nil(t, out.Description)
	assert.Nil(t, out.Backend) // both backend fields empty
}

func TestBuildWorkspaceInput(t *testing.T) {
	req := server.WorkspaceCreateRequest{
		Name:          "ws-1",
		DisplayName:   "WS",
		Description:   "wd",
		Image:         "img:ws",
		ContainerPort: 8888,
		Command:       []string{"jupyter"},
		Args:          []string{"lab"},
		Env:           []server.EnvVar{{Name: "E", Value: "1"}},
		PoolName:      "pool-a",
		UnitName:      "unit-a",
		Quota:         "quota-a",
	}
	out, err := svcutil.BuildWorkspaceInput(req)
	require.NoError(t, err)

	assert.Equal(t, "ws-1", out.Name)
	require.NotNil(t, out.Kind)
	assert.Equal(t, "workspace", *out.Kind)
	assert.Equal(t, "pool-a", out.PoolName)
	assert.Equal(t, "unit-a", out.UnitName)
	assert.Equal(t, "quota-a", out.Quota)

	require.Len(t, out.Roles, 1)
	role := out.Roles[0]
	assert.Equal(t, "workspace", role.Name)
	assert.Equal(t, int32(1), role.Replicas)
	assert.Equal(t, "img:ws", role.Template.Image)
	require.NotNil(t, role.Template.Command)
	assert.Equal(t, []string{"jupyter"}, *role.Template.Command)
	require.NotNil(t, role.Template.Args)
	assert.Equal(t, []string{"lab"}, *role.Template.Args)
	require.NotNil(t, role.Template.Ports)
	require.Len(t, *role.Template.Ports, 1)
	assert.Equal(t, "http", (*role.Template.Ports)[0].Name)
	assert.Equal(t, int32(8888), (*role.Template.Ports)[0].ContainerPort)

	require.NotNil(t, out.Route)
	assert.False(t, out.Route.Enabled)

	require.NotNil(t, out.Annotations)
	annos := *out.Annotations
	assert.Equal(t, "pool-a", annos["resource.axisml.io/pool"])
	assert.Equal(t, "unit-a", annos["resource.axisml.io/unit"])
	// Workspaces carry no model annotations.
	_, hasModel := annos["platform.axisml.io/model-name"]
	assert.False(t, hasModel)
}

func TestBuildWorkspaceInput_NoPortWhenZero(t *testing.T) {
	req := server.WorkspaceCreateRequest{
		Name: "ws-2", Image: "img", PoolName: "p", UnitName: "u",
	}
	out, err := svcutil.BuildWorkspaceInput(req)
	require.NoError(t, err)
	require.Len(t, out.Roles, 1)
	assert.Nil(t, out.Roles[0].Template.Ports)
	assert.Nil(t, out.Roles[0].Template.Command)
	assert.Nil(t, out.DisplayName)
	assert.Nil(t, out.Description)
}

func serviceFixture() *computeservice.MLService {
	id := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	dn, desc, owner, endpoint, msg := "Disp", "A desc", "alice", "https://svc.example", "ready"
	annos := map[string]string{
		"platform.axisml.io/model-name":    "resnet",
		"platform.axisml.io/model-version": "v2",
		"resource.axisml.io/pool":          "pool-a",
		"resource.axisml.io/unit":          "unit-a",
	}
	return &computeservice.MLService{
		Id:          id,
		Namespace:   "ns-acme",
		Name:        "svc-1",
		Phase:       "Ready",
		DisplayName: &dn,
		Description: &desc,
		Owner:       &owner,
		Annotations: &annos,
		CreatedAt:   time.Unix(1699990000, 0).UTC(),
		UpdatedAt:   time.Unix(1699995000, 0).UTC(),
		Spec: gen.MLServiceSpec{
			Scheduling: gen.MLServiceScheduling{Quota: "quota-a"},
			Roles: []gen.MLServiceRoleSpec{{
				Name:     "default",
				Replicas: 2,
				Template: gen.MLServicePodTemplate{
					Image:   "img:svc",
					Command: &[]string{"serve"},
					Args:    &[]string{"--x"},
				},
			}},
			Route: &gen.MLServiceRoute{Enabled: true, Path: sp("/svc")},
		},
		Status: gen.MLServiceStatus{ReadyReplicas: 1, Endpoint: &endpoint, Message: &msg},
	}
}

func TestServiceToView_Full(t *testing.T) {
	s := serviceFixture()
	v := svcutil.ServiceToView(s, "acme")

	assert.Equal(t, server.UUID(s.Id.String()), v.ID)
	assert.Equal(t, "ns-acme", v.Namespace)
	assert.Equal(t, "acme", v.TenantName)
	assert.Equal(t, "svc-1", v.Name)
	assert.Equal(t, "Disp", v.DisplayName)
	assert.Equal(t, "A desc", v.Description)
	assert.Equal(t, "alice", v.Owner)
	assert.Equal(t, "quota-a", v.Quota)
	assert.Equal(t, 1, v.ReadyReplicas)
	assert.Equal(t, server.MLServicePhase("Ready"), v.Phase)
	assert.Equal(t, "ready", v.Message)
	assert.Equal(t, "https://svc.example", v.AccessURL)

	assert.Equal(t, "resnet", v.ModelName)
	assert.Equal(t, "v2", v.ModelVersion)
	assert.Equal(t, "pool-a", v.PoolName)
	assert.Equal(t, "unit-a", v.UnitName)

	assert.Equal(t, 2, v.Replicas)
	assert.Equal(t, "img:svc", v.Image)
	assert.Equal(t, []string{"serve"}, v.Command)
	assert.Equal(t, []string{"--x"}, v.Args)

	assert.True(t, v.Route.Enabled)
	assert.Equal(t, "/svc", v.Route.Path)
}

func TestServiceToView_NilOptionals(t *testing.T) {
	s := &computeservice.MLService{
		Id:        uuid.Nil,
		Namespace: "ns",
		Name:      "svc",
		Phase:     "Creating",
	}
	v := svcutil.ServiceToView(s, "acme")
	assert.Empty(t, v.DisplayName)
	assert.Empty(t, v.Description)
	assert.Empty(t, v.Owner)
	assert.Empty(t, v.ModelName)
	assert.Empty(t, v.PoolName)
	assert.Empty(t, v.UnitName)
	assert.Zero(t, v.Replicas)
	assert.Empty(t, v.Image)
	assert.Nil(t, v.Command)
	assert.False(t, v.Route.Enabled)
}

func TestWorkspaceToView_Running(t *testing.T) {
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	dn, endpoint := "My WS", "https://ws.example"
	annos := map[string]string{
		"resource.axisml.io/pool": "pool-a",
		"resource.axisml.io/unit": "unit-a",
	}
	s := &computeservice.MLService{
		Id:          id,
		Namespace:   "ns-acme",
		Name:        "ws-1",
		Phase:       "Ready",
		DisplayName: &dn,
		Annotations: &annos,
		Spec: gen.MLServiceSpec{
			Scheduling: gen.MLServiceScheduling{Quota: "quota-a"},
			Roles: []gen.MLServiceRoleSpec{{
				Name:     "workspace",
				Replicas: 2,
				Template: gen.MLServicePodTemplate{Image: "img:ws", Command: &[]string{"jupyter"}, Args: &[]string{"lab"}},
			}},
		},
		Status: gen.MLServiceStatus{ReadyReplicas: 2, Endpoint: &endpoint},
	}
	v := svcutil.WorkspaceToView(s, "acme")

	assert.Equal(t, server.UUID(id.String()), v.ID)
	assert.Equal(t, "acme", v.TenantName)
	assert.Equal(t, "My WS", v.DisplayName)
	assert.Equal(t, "quota-a", v.Quota)
	assert.Equal(t, "pool-a", v.PoolName)
	assert.Equal(t, "unit-a", v.UnitName)
	assert.Equal(t, 2, v.Replicas)
	assert.Equal(t, "img:ws", v.Image)
	assert.Equal(t, []string{"jupyter"}, v.Command)
	assert.Equal(t, []string{"lab"}, v.Args)
	assert.Equal(t, "https://ws.example", v.Endpoint.AccessURL)
	assert.Equal(t, server.WorkspacePhase("Ready"), v.Phase)
	assert.Equal(t, server.WorkspaceDesiredState("Running"), v.DesiredState)
}

func TestWorkspaceToView_StoppedWhenZeroReplicas(t *testing.T) {
	s := &computeservice.MLService{
		Id:        uuid.Nil,
		Namespace: "ns",
		Name:      "ws",
		Phase:     "Stopped",
		Spec: gen.MLServiceSpec{
			Roles: []gen.MLServiceRoleSpec{{Name: "workspace", Replicas: 0, Template: gen.MLServicePodTemplate{Image: "img"}}},
		},
	}
	v := svcutil.WorkspaceToView(s, "acme")
	assert.Zero(t, v.Replicas)
	assert.Equal(t, server.WorkspaceDesiredState("Stopped"), v.DesiredState)
}
