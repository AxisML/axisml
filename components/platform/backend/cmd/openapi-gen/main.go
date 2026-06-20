// openapi-gen renders an OpenAPI 3.0.3 description of the AxisML Platform HTTP
// API to docs/openapi/platform.yaml at the repo root.
//
// Platform is not yet implemented; this generator is the contract-only "shell".
// Schemas are derived from the Go DTO structs in internal/api via the shared
// pkg/openapigen reflection engine (the same one cluster-manager /
// compute-service / artifact-hub use), so when the service is built the spec
// will already be in lock-step with its request/response types. Routes are
// listed explicitly in paths.go (single source of truth) rather than scraped
// from a router so the file stays reviewable.
//
// Run from the component root:
//
//	go run ./cmd/openapi-gen -o ../../../docs/openapi/platform.yaml
//
// Limitations vs the hand-written design spec: the openapigen engine models
// only `components/schemas` (not `components/parameters` / `responses` /
// `securitySchemes`), and does not carry field/schema descriptions or `default`
// values. Shared parameters/responses are therefore inlined per operation and
// the bearer-JWT security scheme is omitted — matching the other components'
// generated specs.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/axisml/axisml/components/platform/backend/internal/server"
	"github.com/axisml/axisml/pkg/openapigen"
)

const defaultVersion = "0.0.0-dev"

// Tag names. One source of truth so a typo can't silently split a group.
const (
	tagAuth          = "Auth"
	tagUsers         = "Users"
	tagTenants       = "Tenants"
	tagQuotas        = "Quotas"
	tagMembers       = "Members"
	tagWorkspaces    = "Workspaces"
	tagExperiments   = "Experiments"
	tagJobs          = "Jobs"
	tagMLServices    = "MLServices"
	tagTrafficPolicy = "TrafficPolicy"
	tagModels        = "Models"
	tagImages        = "Images"
	tagResourcePools = "ResourcePools"
	tagResourceUnits = "ResourceUnits"
	tagDashboard     = "Dashboard"
	tagAudit         = "Audit"
	tagHealth        = "Health"
)

// Name/version patterns. Duplicated here as regexes (rather than imported from
// a strutil package) because the generator's contract is "render whatever
// clients must send"; clients don't import our Go helpers.
const (
	dns1123Pattern         = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	artifactNamePattern    = "^[a-z0-9]([-a-z0-9._]*[a-z0-9])?$"
	artifactVersionPattern = "^[A-Za-z0-9_.-]{1,128}$"
)

func main() {
	out := flag.String("o", "../../../docs/openapi/platform.yaml", "output path")
	v := flag.String("version", defaultVersion, "info.version field")
	flag.Parse()

	doc := buildDocument(*v)
	data, err := openapigen.MarshalYAML(doc)
	if err != nil {
		fail("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fail("write: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "openapi-gen: "+format+"\n", args...)
	os.Exit(1)
}

func buildDocument(version string) *openapigen.Document {
	g := openapigen.New(openapigen.Options{
		WellKnown: wellKnown,
		PatternRules: []openapigen.PatternRule{
			{Tag: "dns1123", Pattern: dns1123Pattern},
			{Tag: "artifactname", Pattern: artifactNamePattern},
		},
		// All DTOs live in internal/api; map that package to an empty prefix so
		// each component schema's name is exactly its Go type name (and nested
		// $refs resolve to the same names).
		PackageNamer: func(pkg string) (string, bool) {
			if strings.HasSuffix(pkg, "/components/platform/backend/internal/server") {
				return "", true
			}
			return "", false
		},
	})

	registerSchemas(g)

	return &openapigen.Document{
		OpenAPI: "3.0.3",
		Info: openapigen.Info{
			Title:   "AxisML Platform API",
			Version: version,
			Description: "User-facing entrypoint to AxisML. Aggregates the internal services " +
				"(cluster-manager / compute / artifacts) and adds platform-native features: " +
				"identity, RBAC, workspace/job/experiment/service/traffic orchestration, the " +
				"model/image artifact registry, and Prometheus-backed dashboards. All endpoints except " +
				"POST /api/v1/auth/login require a bearer JWT issued by Platform; per-endpoint " +
				"role requirements are noted in each operation. Errors use RFC 7807 " +
				"application/problem+json bodies.",
		},
		Servers: []openapigen.ServerEntry{
			{URL: "/", Description: "Same-origin (behind the AxisML Infra Envoy Gateway)"},
		},
		Tags:  tags(),
		Paths: paths(),
		Components: openapigen.ComponentsBlock{
			Schemas: g.Schemas(),
		},
	}
}

func tags() []openapigen.TagEntry {
	return []openapigen.TagEntry{
		{Name: tagAuth, Description: "Login, logout, session refresh, and \"who am I\" endpoints."},
		{Name: tagUsers, Description: "Platform user accounts (the identities used for login)."},
		{Name: tagTenants, Description: "Tenant lifecycle. Proxies cluster-manager with RBAC."},
		{Name: tagQuotas, Description: "ElasticQuota sub-resources of a Tenant (per-pool, per-quota)."},
		{Name: tagMembers, Description: "Per-tenant user ↔ role bindings."},
		{Name: tagWorkspaces, Description: "Long-running interactive dev containers (jupyter / code-server / ...)."},
		{Name: tagExperiments, Description: "Training-experiment templates (Platform-owned) and their Runs; isomorphic to Jobs, plus on-demand TensorBoard."},
		{Name: tagJobs, Description: "Reusable Job templates (Platform-owned definitions) and their Runs (compute MLRuns named <job>-<n>)."},
		{Name: tagMLServices, Description: "Long-running online inference services."},
		{Name: tagTrafficPolicy, Description: "Traffic policies fanning one stable entry across member online services (weighted / canary)."},
		{Name: tagModels, Description: "Model definitions (Platform-owned); versions proxy artifacts (kind=model)."},
		{Name: tagImages, Description: "Image definitions (Platform-owned); versions proxy artifacts (kind=image)."},
		{Name: tagResourcePools, Description: "Cluster-scoped resource pools (admin-only writes; read by all)."},
		{Name: tagResourceUnits, Description: "Per-pool resource unit specs (admin-only writes; read by all)."},
		{Name: tagDashboard, Description: "Aggregated overview and Prometheus-backed metrics."},
		{Name: tagAudit, Description: "Audit log queries (system-admin only)."},
		{Name: tagHealth, Description: "Liveness and readiness probes."},
	}
}

// registerSchemas declares every component schema. Struct DTOs are reflected
// (InputMode for request bodies, ResponseMode for everything else); named
// enums / maps / free-form specs are set directly because they are surfaced via
// the WellKnown $ref hook rather than reflected from a struct.
func registerSchemas(g *openapigen.Generator) {
	// --- Request-body DTOs (InputMode) ---
	for name, v := range map[string]any{
		"LoginRequest":                  server.LoginRequest{},
		"UserCreateRequest":             server.UserCreateRequest{},
		"UserPatchRequest":              server.UserPatchRequest{},
		"SetPasswordRequest":            server.SetPasswordRequest{},
		"QuotaCreateRequest":            server.QuotaCreateRequest{},
		"QuotaPatchRequest":             server.QuotaPatchRequest{},
		"TenantCreateRequest":           server.TenantCreateRequest{},
		"TenantPatchRequest":            server.TenantPatchRequest{},
		"MemberCreateRequest":           server.MemberCreateRequest{},
		"MemberPatchRequest":            server.MemberPatchRequest{},
		"WorkspaceCreateRequest":        server.WorkspaceCreateRequest{},
		"WorkspacePatchRequest":         server.WorkspacePatchRequest{},
		"WorkspaceDeleteRequest":        server.WorkspaceDeleteRequest{},
		"JobCreateInput":                server.JobCreateInput{},
		"JobPatchInput":                 server.JobPatchInput{},
		"RunTriggerInput":               server.RunTriggerInput{},
		"MLServiceCreateRequest":        server.MLServiceCreateRequest{},
		"MLServicePatchRequest":         server.MLServicePatchRequest{},
		"MLServiceScaleRequest":         server.MLServiceScaleRequest{},
		"ModelInitiateRequest":          server.ModelInitiateRequest{},
		"ModelCompleteRequest":          server.ModelCompleteRequest{},
		"ArtifactUpdateRequest":         server.ArtifactUpdateRequest{},
		"ArtifactDefinitionCreateInput": server.ArtifactDefinitionCreateInput{},
		"ArtifactDefinitionPatchInput":  server.ArtifactDefinitionPatchInput{},
		"ImageInitiateRequest":          server.ImageInitiateRequest{},
		"ImageCompleteRequest":          server.ImageCompleteRequest{},
		"ResourcePoolCreateRequest":     server.ResourcePoolCreateRequest{},
		"ResourcePoolPatchRequest":      server.ResourcePoolPatchRequest{},
		"ResourceUnitCreateRequest":     server.ResourceUnitCreateRequest{},
		"ResourceUnitPatchRequest":      server.ResourceUnitPatchRequest{},
		"ExperimentCreateInput":         server.ExperimentCreateInput{},
		"ExperimentPatchInput":          server.ExperimentPatchInput{},
		"TensorBoardRequest":            server.TensorBoardRequest{},
		"TrafficPolicyCreateRequest":    server.TrafficPolicyCreateRequest{},
		"TrafficPolicyPatchRequest":     server.TrafficPolicyPatchRequest{},
		"TrafficPolicySplitRequest":     server.TrafficPolicySplitRequest{},
		"TrafficPolicyBackendInput":     server.TrafficPolicyBackendInput{},
	} {
		g.Register(name, v, openapigen.InputMode)
	}

	// --- Response / nested DTOs (ResponseMode) ---
	for name, v := range map[string]any{
		"Problem":                 server.Problem{},
		"ProblemFieldError":       server.ProblemFieldError{},
		"HealthStatus":            server.HealthStatus{},
		"EnvVar":                  server.EnvVar{},
		"Condition":               server.Condition{},
		"LoginResponse":           server.LoginResponse{},
		"RefreshResponse":         server.RefreshResponse{},
		"UserTenantRole":          server.UserTenantRole{},
		"MeResponse":              server.MeResponse{},
		"User":                    server.User{},
		"UserSummary":             server.UserSummary{},
		"UserSummaryList":         server.UserSummaryList{},
		"Quota":                   server.Quota{},
		"QuotaUnit":               server.QuotaUnit{},
		"QuotaUnitStatus":         server.QuotaUnitStatus{},
		"QuotaStatus":             server.QuotaStatus{},
		"QuotaList":               server.QuotaList{},
		"SecretSourceRef":         server.SecretSourceRef{},
		"ConfigMapSourceRef":      server.ConfigMapSourceRef{},
		"ImagePullSecretInit":     server.ImagePullSecretInit{},
		"SecretInit":              server.SecretInit{},
		"ConfigMapInit":           server.ConfigMapInit{},
		"ServiceAccountInit":      server.ServiceAccountInit{},
		"InitResources":           server.InitResources{},
		"Tenant":                  server.Tenant{},
		"TenantStatus":            server.TenantStatus{},
		"TenantList":              server.TenantList{},
		"Member":                  server.Member{},
		"MemberList":              server.MemberList{},
		"WorkspaceLifecycle":      server.WorkspaceLifecycle{},
		"WorkspaceVolume":         server.WorkspaceVolume{},
		"WorkspaceEndpoint":       server.WorkspaceEndpoint{},
		"Workspace":               server.Workspace{},
		"WorkspaceList":           server.WorkspaceList{},
		"WorkspaceAccess":         server.WorkspaceAccess{},
		"Backend":                 server.Backend{},
		"RoleTemplate":            server.RoleTemplate{},
		"MLRunRole":               server.MLRunRole{},
		"MLRunRoleStatus":         server.MLRunRoleStatus{},
		"RunPolicy":               server.RunPolicy{},
		"RunView":                 server.RunView{},
		"MLRunSpec":               server.MLRunSpec{},
		"RunList":                 server.RunList{},
		"ArtifactRef":             server.ArtifactRef{},
		"JobSpec":                 server.JobSpec{},
		"JobView":                 server.JobView{},
		"JobList":                 server.JobList{},
		"ServicePort":             server.ServicePort{},
		"MLServiceRoute":          server.MLServiceRoute{},
		"MLService":               server.MLService{},
		"MLServiceList":           server.MLServiceList{},
		"MetricPoint":             server.MetricPoint{},
		"MetricSeries":            server.MetricSeries{},
		"Model":                   server.Model{},
		"ModelList":               server.ModelList{},
		"ModelInitiateResponse":   server.ModelInitiateResponse{},
		"ArtifactDefinitionView":  server.ArtifactDefinitionView{},
		"ArtifactDefinitionList":  server.ArtifactDefinitionList{},
		"ArtifactResolveResponse": server.ArtifactResolveResponse{},
		"Image":                   server.Image{},
		"ImageList":               server.ImageList{},
		"ImageInitiateResponse":   server.ImageInitiateResponse{},
		"ExperimentView":          server.ExperimentView{},
		"ExperimentList":          server.ExperimentList{},
		"TensorBoard":             server.TensorBoard{},
		"TrafficPolicyEndpoint":   server.TrafficPolicyEndpoint{},
		"TrafficPolicyBackend":    server.TrafficPolicyBackend{},
		"TrafficPolicy":           server.TrafficPolicy{},
		"TrafficPolicyList":       server.TrafficPolicyList{},
		"ResourcePool":            server.ResourcePool{},
		"ResourcePoolList":        server.ResourcePoolList{},
		"ResourceUnit":            server.ResourceUnit{},
		"ResourceUnitList":        server.ResourceUnitList{},
		"Pod":                     server.Pod{},
		"PodList":                 server.PodList{},
		"Event":                   server.Event{},
		"EventList":               server.EventList{},
		"DashboardOverview":       server.DashboardOverview{},
		"AuditLog":                server.AuditLog{},
		"AuditLogList":            server.AuditLogList{},
	} {
		g.Register(name, v, openapigen.ResponseMode)
	}

	// --- Named enums (referenced via $ref) ---
	g.Set("RoleName", enumSchema(server.RoleNameValues, ""))
	g.Set("TenantPhase", enumSchema(server.TenantPhaseValues, ""))
	g.Set("WorkspacePhase", enumSchema(server.WorkspacePhaseValues,
		"Derived from compute service phase + replicas. Hoisted to the top of Workspace for B-tree filtering."))
	g.Set("WorkspaceDesiredState", enumSchema(server.WorkspaceDesiredStateValues, ""))
	g.Set("RunPhase", enumSchema(server.RunPhaseValues,
		"Run (compute MLRun) phase. The active (non-terminal) phases — Creating / Pending / Running / Canceling — block Job-definition deletion."))
	g.Set("MLServicePhase", enumSchema(server.MLServicePhaseValues, ""))
	g.Set("MLServiceMetricName", enumSchema(server.MLServiceMetricNameValues, ""))
	g.Set("WorkloadMetricName", enumSchema(server.WorkloadMetricNameValues, "Run / workload resource metric."))
	g.Set("ModelStatus", enumSchema(server.ArtifactStatusValues, "Mirrors artifacts ArtifactStatus for kind=model."))
	g.Set("ImageStatus", enumSchema(server.ArtifactStatusValues, "Mirrors artifacts ArtifactStatus for kind=image."))
	g.Set("ImagePurpose", enumSchema(server.ImagePurposeValues, "Image definition intended use (list filter)."))
	g.Set("ArtifactSource", enumSchema(server.ArtifactSourceValues, "How an artifact version was added."))
	g.Set("RemoteSourceKind", enumSchema(server.RemoteSourceKindValues, "Backing store of an externally-registered model version."))
	g.Set("TrafficPolicyMode", enumSchema(server.TrafficPolicyModeValues, ""))
	g.Set("TrafficPolicyBackendRole", enumSchema(server.TrafficPolicyBackendRoleValues, ""))
	g.Set("TrafficPolicyPhase", enumSchema(server.TrafficPolicyPhaseValues, ""))
	g.Set("TensorBoardPhase", enumSchema(server.TensorBoardPhaseValues, ""))

	// --- Named maps / free-form specs (referenced via $ref) ---
	g.Set("StringMap", &openapigen.Schema{Type: "object", AdditionalProperties: &openapigen.Schema{Type: "string"}})
	g.Set("ResourceMap", &openapigen.Schema{
		Type:                 "object",
		Description:          "Kubernetes-style resource quantity map (e.g., {\"cpu\": \"100\", \"memory\": \"1Ti\", \"nvidia.com/gpu\": \"8\"}).",
		AdditionalProperties: &openapigen.Schema{Type: "string"},
	})
	g.Set("Toleration", &openapigen.Schema{
		Type:                 "object",
		Description:          "Mirrors a Kubernetes corev1.Toleration.",
		AdditionalProperties: &openapigen.Schema{},
	})
	g.Set("ModelSpec", freeFormSpec("Artifact-side spec for kind=model; pass-through to artifacts."))
	g.Set("ImageSpec", freeFormSpec("Artifact-side spec for kind=image; pass-through to artifacts."))
}

func enumSchema(values []string, desc string) *openapigen.Schema {
	return &openapigen.Schema{Type: "string", Description: desc, Enum: append([]string(nil), values...)}
}

func freeFormSpec(desc string) *openapigen.Schema {
	return &openapigen.Schema{Type: "object", Description: desc, AdditionalProperties: &openapigen.Schema{}}
}

// --- WellKnown: scalar formats, named enums/maps as $refs, inline enums ---

var (
	tUUID     = reflect.TypeOf(server.UUID(""))
	tEmail    = reflect.TypeOf(server.Email(""))
	tPassword = reflect.TypeOf(server.Password(""))
	tURI      = reflect.TypeOf(server.URI(""))

	tStringMap   = reflect.TypeOf(server.StringMap(nil))
	tResourceMap = reflect.TypeOf(server.ResourceMap(nil))
	tToleration  = reflect.TypeOf(server.Toleration(nil))
	tModelSpec   = reflect.TypeOf(server.ModelSpec(nil))
	tImageSpec   = reflect.TypeOf(server.ImageSpec(nil))

	tRoleName              = reflect.TypeOf(server.RoleName(""))
	tTenantPhase           = reflect.TypeOf(server.TenantPhase(""))
	tWorkspacePhase        = reflect.TypeOf(server.WorkspacePhase(""))
	tWorkspaceDesiredState = reflect.TypeOf(server.WorkspaceDesiredState(""))
	tRunPhase              = reflect.TypeOf(server.RunPhase(""))
	tMLServicePhase        = reflect.TypeOf(server.MLServicePhase(""))
	tMLServiceMetricName   = reflect.TypeOf(server.MLServiceMetricName(""))
	tModelStatus           = reflect.TypeOf(server.ModelStatus(""))
	tImageStatus           = reflect.TypeOf(server.ImageStatus(""))

	tArtifactSource           = reflect.TypeOf(server.ArtifactSource(""))
	tRemoteSourceKind         = reflect.TypeOf(server.RemoteSourceKind(""))
	tTrafficPolicyMode        = reflect.TypeOf(server.TrafficPolicyMode(""))
	tTrafficPolicyBackendRole = reflect.TypeOf(server.TrafficPolicyBackendRole(""))
	tTrafficPolicyPhase       = reflect.TypeOf(server.TrafficPolicyPhase(""))
	tTensorBoardPhase         = reflect.TypeOf(server.TensorBoardPhase(""))

	tHealthState     = reflect.TypeOf(server.HealthState(""))
	tConditionStatus = reflect.TypeOf(server.ConditionStatus(""))
	tMemberRoleName  = reflect.TypeOf(server.MemberRoleName(""))
	tBackendName     = reflect.TypeOf(server.BackendName(""))
	tRestartPolicy   = reflect.TypeOf(server.RestartPolicy(""))
	tArtifactKind    = reflect.TypeOf(server.ArtifactKind(""))
	tStorageKind     = reflect.TypeOf(server.StorageKind(""))
	tDefinitionKind  = reflect.TypeOf(server.DefinitionKind(""))
	tPodPhase        = reflect.TypeOf(server.PodPhase(""))
	tEventType       = reflect.TypeOf(server.EventType(""))
	tAuditResult     = reflect.TypeOf(server.AuditResult(""))
)

// wellKnown returns a freshly-allocated schema each call so that per-field
// validators (e.g. a Password with binding min=8) never mutate a shared value.
func wellKnown(t reflect.Type) *openapigen.Schema {
	switch t {
	case tUUID:
		return &openapigen.Schema{Type: "string", Format: "uuid"}
	case tEmail:
		return &openapigen.Schema{Type: "string", Format: "email"}
	case tPassword:
		return &openapigen.Schema{Type: "string", Format: "password"}
	case tURI:
		return &openapigen.Schema{Type: "string", Format: "uri"}

	case tStringMap:
		return openapigen.Ref("StringMap")
	case tResourceMap:
		return openapigen.Ref("ResourceMap")
	case tToleration:
		return openapigen.Ref("Toleration")
	case tModelSpec:
		return openapigen.Ref("ModelSpec")
	case tImageSpec:
		return openapigen.Ref("ImageSpec")

	case tRoleName:
		return openapigen.Ref("RoleName")
	case tTenantPhase:
		return openapigen.Ref("TenantPhase")
	case tWorkspacePhase:
		return openapigen.Ref("WorkspacePhase")
	case tWorkspaceDesiredState:
		return openapigen.Ref("WorkspaceDesiredState")
	case tRunPhase:
		return openapigen.Ref("RunPhase")
	case tMLServicePhase:
		return openapigen.Ref("MLServicePhase")
	case tMLServiceMetricName:
		return openapigen.Ref("MLServiceMetricName")
	case tModelStatus:
		return openapigen.Ref("ModelStatus")
	case tImageStatus:
		return openapigen.Ref("ImageStatus")

	case tArtifactSource:
		return openapigen.Ref("ArtifactSource")
	case tRemoteSourceKind:
		return openapigen.Ref("RemoteSourceKind")
	case tTrafficPolicyMode:
		return openapigen.Ref("TrafficPolicyMode")
	case tTrafficPolicyBackendRole:
		return openapigen.Ref("TrafficPolicyBackendRole")
	case tTrafficPolicyPhase:
		return openapigen.Ref("TrafficPolicyPhase")
	case tTensorBoardPhase:
		return openapigen.Ref("TensorBoardPhase")

	case tHealthState:
		return enumSchema(server.HealthStateValues, "")
	case tConditionStatus:
		return enumSchema(server.ConditionStatusValues, "")
	case tMemberRoleName:
		return enumSchema(server.MemberRoleNameValues, "")
	case tBackendName:
		return enumSchema(server.BackendNameValues, "")
	case tRestartPolicy:
		return enumSchema(server.RestartPolicyValues, "")
	case tArtifactKind:
		return enumSchema(server.ArtifactKindValues, "")
	case tStorageKind:
		return enumSchema(server.StorageKindValues, "")
	case tDefinitionKind:
		return enumSchema(server.DefinitionKindValues, "")
	case tPodPhase:
		return enumSchema(server.PodPhaseValues, "")
	case tEventType:
		return enumSchema(server.EventTypeValues, "")
	case tAuditResult:
		return enumSchema(server.AuditResultValues, "")
	}
	return nil
}
