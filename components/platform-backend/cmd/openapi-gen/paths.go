package main

import (
	"strings"

	"github.com/axisml/axisml/pkg/openapigen"
)

// ---- response helpers -------------------------------------------------------

// errDescriptions pins the standard problem-response descriptions, keyed by
// status code. Mirrors the design spec's components/responses block.
var errDescriptions = map[string]string{
	"400": "Malformed request (invalid syntax, missing required fields, immutable field, etc.).",
	"401": "Missing or invalid bearer JWT.",
	"403": "Authenticated but insufficient role for the requested operation.",
	"404": "Resource not found, OR the caller cannot see it. Platform deliberately conflates these for non-admins to avoid information leak.",
	"409": "Conflicts with a precondition: duplicate name, last-tenant-admin protection, in-use blocker, immutable transition, etc.",
	"410": "Target resource (e.g., a Pod log) is no longer retained.",
	"422": "Semantically invalid request (e.g., cross-field violations).",
	"500": "Unexpected server error.",
	"502": "Upstream service (cluster-manager / compute / artifacts / Prometheus) failed.",
	"503": "One or more dependencies are unavailable.",
}

// problemResp builds an application/problem+json response referencing Problem.
func problemResp(desc string) openapigen.Response {
	return openapigen.Response{
		Description: desc,
		Content:     map[string]openapigen.MediaType{"application/problem+json": {Schema: openapigen.Ref("Problem")}},
	}
}

func errResp(code string) openapigen.Response { return problemResp(errDescriptions[code]) }

// sc builds a single success response. ref == "" yields a no-body response (204).
func sc(code, desc, ref string) map[string]openapigen.Response {
	if ref == "" {
		return map[string]openapigen.Response{code: {Description: desc}}
	}
	return map[string]openapigen.Response{code: openapigen.JSONResp(desc, ref)}
}

// resp overlays the standard problem responses for the given codes onto a
// success map.
func resp(success map[string]openapigen.Response, errCodes ...string) map[string]openapigen.Response {
	out := make(map[string]openapigen.Response, len(success)+len(errCodes))
	for k, v := range success {
		out[k] = v
	}
	for _, c := range errCodes {
		out[c] = errResp(c)
	}
	return out
}

func logStream() map[string]openapigen.Response {
	s := &openapigen.Schema{Type: "string"}
	return map[string]openapigen.Response{"200": {
		Description: "Log stream. text/plain for follow=false; text/event-stream for follow=true.",
		Content: map[string]openapigen.MediaType{
			"text/plain":        {Schema: s},
			"text/event-stream": {Schema: s},
		},
	}}
}

func body(ref string) *openapigen.RequestBody { return openapigen.JSONBody(ref) }

func optBody(ref string) *openapigen.RequestBody {
	b := openapigen.JSONBody(ref)
	b.Required = false
	return b
}

func newOp(tag, id, summary string, params []openapigen.Parameter, reqBody *openapigen.RequestBody, responses map[string]openapigen.Response) *openapigen.Operation {
	return &openapigen.Operation{
		Tags:        []string{tag},
		OperationID: id,
		Summary:     summary,
		Parameters:  params,
		RequestBody: reqBody,
		Responses:   responses,
	}
}

// ---- parameter helpers ------------------------------------------------------

func fptr(v float64) *float64 { return &v }
func iptr(v int) *int         { return &v }

func strSchema() *openapigen.Schema  { return &openapigen.Schema{Type: "string"} }
func boolSchema() *openapigen.Schema { return &openapigen.Schema{Type: "boolean"} }
func strEnum(vals ...string) *openapigen.Schema {
	return &openapigen.Schema{Type: "string", Enum: vals}
}

func qp(name, desc string, schema *openapigen.Schema) openapigen.Parameter {
	return openapigen.Parameter{Name: name, In: "query", Description: desc, Schema: schema}
}

func qpReq(name, desc string, schema *openapigen.Schema) openapigen.Parameter {
	p := qp(name, desc, schema)
	p.Required = true
	return p
}

func pathParam(name, pattern string, min, max int, format string) openapigen.Parameter {
	s := &openapigen.Schema{Type: "string"}
	if pattern != "" {
		s.Pattern = pattern
	}
	if min > 0 {
		s.MinLength = iptr(min)
	}
	if max > 0 {
		s.MaxLength = iptr(max)
	}
	if format != "" {
		s.Format = format
	}
	return openapigen.Parameter{Name: name, In: "path", Required: true, Schema: s}
}

func activeTenant(required bool) openapigen.Parameter {
	return openapigen.Parameter{
		Name: "X-Axisml-Tenant", In: "header", Required: required,
		Description: "Active tenant for the request.",
		Schema:      &openapigen.Schema{Type: "string", Pattern: dns1123Pattern, MinLength: iptr(3), MaxLength: iptr(40)},
	}
}

// path-parameter constructors (one per distinct shape).
func tenantNameP() openapigen.Parameter {
	return pathParam("name", dns1123Pattern, 3, 40, "Tenant identifier and logical tenant scope; not a Kubernetes Namespace.")
}
func tenantP() openapigen.Parameter            { return pathParam("tenant", dns1123Pattern, 3, 40, "") }
func quotaPoolP() openapigen.Parameter         { return pathParam("pool", dns1123Pattern, 1, 63, "") }
func userIDP() openapigen.Parameter            { return pathParam("id", "", 0, 0, "uuid") }
func memberUserIDP() openapigen.Parameter      { return pathParam("userId", "", 0, 0, "uuid") }
func workspaceNameP() openapigen.Parameter     { return pathParam("name", dns1123Pattern, 1, 63, "") }
func mlserviceNameP() openapigen.Parameter     { return pathParam("name", dns1123Pattern, 1, 63, "") }
func experimentNameP() openapigen.Parameter    { return pathParam("name", dns1123Pattern, 1, 63, "") }
func trafficPolicyNameP() openapigen.Parameter { return pathParam("name", dns1123Pattern, 1, 40, "") }
func jobNameP() openapigen.Parameter           { return pathParam("name", dns1123Pattern, 1, 63, "") }
func runNameP() openapigen.Parameter           { return pathParam("run", dns1123Pattern, 1, 63, "") }
func artifactNameP() openapigen.Parameter      { return pathParam("name", artifactNamePattern, 1, 128, "") }
func artifactVerP() openapigen.Parameter {
	return pathParam("version", artifactVersionPattern, 1, 128, "")
}
func poolNameP() openapigen.Parameter { return pathParam("pool", dns1123Pattern, 1, 63, "") }
func unitNameP() openapigen.Parameter { return pathParam("unit", dns1123Pattern, 1, 63, "") }
func podNameP() openapigen.Parameter  { return pathParam("pod", "", 1, 253, "") }

var (
	limitParam    = qp("limit", "Page size (default 50, max 200).", &openapigen.Schema{Type: "integer", Minimum: fptr(1), Maximum: fptr(200)})
	continueParam = qp("continue", "Opaque continue token from the previous response.", strSchema())
	qParam        = qp("q", "Fuzzy keyword.", strSchema())
	ownerParam    = qp("owner", "Admin-only owner filter.", strSchema())
)

func logParams() []openapigen.Parameter {
	return []openapigen.Parameter{
		qp("container", "", strSchema()),
		qp("tailLines", "", &openapigen.Schema{Type: "integer", Minimum: fptr(1)}),
		qp("follow", "Stream logs as SSE.", boolSchema()),
		qp("previous", "Read logs from the previous (crashed) container instance.", boolSchema()),
	}
}

func runMetricsParams() []openapigen.Parameter {
	return []openapigen.Parameter{
		qpReq("metric", "", openapigen.Ref("WorkloadMetricName")),
		qpReq("range", "ISO 8601 duration (5m, 1h, 24h).", strSchema()),
		qp("step", "", strSchema()),
	}
}

// paths declares every route. Path-level parameters in the design spec are
// attached to each operation here (openapigen has no path-level parameter slot).
func paths() map[string]openapigen.PathItem {
	p := map[string]openapigen.PathItem{}

	// ---- Health ----
	p["/healthz"] = openapigen.PathItem{Get: newOp(tagHealth, "getHealthz", "Liveness probe",
		nil, nil, resp(sc("200", "Process is alive.", "HealthStatus"), "500"))}
	p["/readyz"] = openapigen.PathItem{Get: newOp(tagHealth, "getReadyz", "Readiness probe",
		nil, nil, resp(sc("200", "Service is ready to accept traffic.", "HealthStatus"), "503"))}

	// ---- Auth ----
	p["/api/v1/auth/login"] = openapigen.PathItem{Post: newOp(tagAuth, "login", "Exchange username + password for a bearer JWT",
		nil, body("LoginRequest"), resp(sc("200", "Login succeeded; returns JWT and current user.", "LoginResponse"), "400", "401", "422", "500"))}
	p["/api/v1/auth/logout"] = openapigen.PathItem{Post: newOp(tagAuth, "logout", "Revoke the current bearer JWT",
		nil, nil, resp(sc("204", "Session revoked.", ""), "401", "500"))}
	p["/api/v1/auth/me"] = openapigen.PathItem{Get: newOp(tagAuth, "getCurrentUser", "Get the current user, tenant role bindings, and permissions",
		nil, nil, resp(sc("200", "Current user view.", "MeResponse"), "401", "500"))}
	p["/api/v1/auth/refresh"] = openapigen.PathItem{Post: newOp(tagAuth, "refreshToken", "Refresh the current bearer JWT",
		nil, nil, resp(sc("200", "New JWT issued.", "RefreshResponse"), "401", "500"))}

	// ---- Users ----
	p["/api/v1/users"] = openapigen.PathItem{
		Get: newOp(tagUsers, "listUsers", "Search Platform users (system-admin only)",
			[]openapigen.Parameter{qParam, limitParam, continueParam},
			nil, resp(sc("200", "A page of user summaries.", "UserSummaryList"), "401", "403", "500")),
		Post: newOp(tagUsers, "createUser", "Create a Platform user",
			nil, body("UserCreateRequest"), resp(sc("201", "User created.", "User"), "400", "401", "403", "409", "422", "500")),
	}
	p["/api/v1/users/{id}"] = openapigen.PathItem{
		Get: newOp(tagUsers, "getUser", "Get a user by id (system-admin only)",
			[]openapigen.Parameter{userIDP()}, nil, resp(sc("200", "User detail.", "User"), "401", "403", "404", "500")),
		Patch: newOp(tagUsers, "updateUser", "Update a user's profile or disabled flag",
			[]openapigen.Parameter{userIDP()}, body("UserPatchRequest"), resp(sc("200", "Updated user.", "User"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagUsers, "deleteUser", "Delete (or disable) a user",
			[]openapigen.Parameter{userIDP()}, nil, resp(sc("204", "User deleted.", ""), "401", "403", "404", "500")),
	}
	p["/api/v1/users/{id}/password"] = openapigen.PathItem{Post: newOp(tagUsers, "setUserPassword", "Set a user's password",
		[]openapigen.Parameter{userIDP()}, body("SetPasswordRequest"), resp(sc("204", "Password updated.", ""), "400", "401", "403", "404", "422", "500"))}

	// ---- Tenants ----
	p["/api/v1/tenants"] = openapigen.PathItem{
		Post: newOp(tagTenants, "createTenant", "Create a tenant",
			nil, body("TenantCreateRequest"), resp(sc("201", "Tenant created.", "Tenant"), "400", "401", "403", "409", "422", "500")),
		Get: newOp(tagTenants, "listTenants", "List visible tenants",
			[]openapigen.Parameter{
				qp("q", "Fuzzy keyword on name / displayName.", strSchema()),
				qp("phase", "Filter by tenant status.phase.", openapigen.Ref("TenantPhase")),
				limitParam, continueParam,
			}, nil, resp(sc("200", "A page of visible tenants.", "TenantList"), "401", "500")),
	}
	p["/api/v1/tenants/{name}"] = openapigen.PathItem{
		Get: newOp(tagTenants, "getTenant", "Get a tenant",
			[]openapigen.Parameter{tenantNameP()},
			nil, resp(sc("200", "Tenant detail.", "Tenant"), "401", "403", "404", "500")),
		Patch: newOp(tagTenants, "updateTenant", "Update tenant display metadata",
			[]openapigen.Parameter{tenantNameP()}, body("TenantPatchRequest"), resp(sc("200", "Updated tenant.", "Tenant"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagTenants, "deleteTenant", "Delete a tenant",
			[]openapigen.Parameter{tenantNameP()}, nil, resp(sc("204", "Tenant deleted.", ""), "401", "403", "404", "409", "500")),
	}
	p["/api/v1/tenants/{name}/suspend"] = openapigen.PathItem{Post: newOp(tagTenants, "suspendTenant", "Suspend a tenant (gates new-workload creation)",
		[]openapigen.Parameter{tenantNameP()}, nil, resp(sc("200", "Tenant suspended.", "Tenant"), "401", "403", "404", "409", "500"))}
	p["/api/v1/tenants/{name}/resume"] = openapigen.PathItem{Post: newOp(tagTenants, "resumeTenant", "Resume a suspended tenant",
		[]openapigen.Parameter{tenantNameP()}, nil, resp(sc("200", "Tenant resumed.", "Tenant"), "401", "403", "404", "409", "500"))}

	// ---- Quotas ----
	p["/api/v1/tenants/{name}/quotas"] = openapigen.PathItem{
		Get: newOp(tagQuotas, "listTenantQuotas", "List per-pool quotas for a tenant",
			[]openapigen.Parameter{tenantNameP()}, nil, resp(sc("200", "Quota list.", "QuotaList"), "401", "403", "404", "500")),
		Post: newOp(tagQuotas, "createTenantQuota", "Set a pool's quota (system-admin only)",
			[]openapigen.Parameter{tenantNameP()}, body("QuotaCreateRequest"), resp(sc("201", "Quota set.", "Tenant"), "400", "401", "403", "404", "409", "422", "500")),
	}
	p["/api/v1/tenants/{name}/quotas/{pool}"] = openapigen.PathItem{
		Patch: newOp(tagQuotas, "updateTenantQuota", "Update a pool quota (system-admin only)",
			[]openapigen.Parameter{tenantNameP(), quotaPoolP()}, body("QuotaPatchRequest"),
			resp(sc("200", "Quota updated; returns the updated tenant.", "Tenant"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagQuotas, "deleteTenantQuota", "Remove a pool quota (system-admin only)",
			[]openapigen.Parameter{tenantNameP(), quotaPoolP()}, nil,
			resp(sc("204", "Quota removed.", ""), "401", "403", "404", "409", "500")),
	}

	// ---- Members ----
	p["/api/v1/tenants/{name}/members"] = openapigen.PathItem{
		Get: newOp(tagMembers, "listTenantMembers", "List members of a tenant",
			[]openapigen.Parameter{tenantNameP()}, nil, resp(sc("200", "Member list.", "MemberList"), "401", "403", "404", "500")),
		Post: newOp(tagMembers, "addTenantMember", "Add a member to a tenant",
			[]openapigen.Parameter{tenantNameP()}, body("MemberCreateRequest"), resp(sc("201", "Member added.", "Member"), "400", "401", "403", "404", "409", "422", "500")),
	}
	p["/api/v1/tenants/{name}/members/{userId}"] = openapigen.PathItem{
		Patch: newOp(tagMembers, "updateTenantMember", "Change a member's role",
			[]openapigen.Parameter{tenantNameP(), memberUserIDP()}, body("MemberPatchRequest"),
			resp(sc("200", "Member updated.", "Member"), "400", "401", "403", "404", "409", "500")),
		Delete: newOp(tagMembers, "removeTenantMember", "Remove a member from a tenant",
			[]openapigen.Parameter{tenantNameP(), memberUserIDP()}, nil,
			resp(sc("204", "Member removed.", ""), "401", "403", "404", "409", "500")),
	}

	// ---- Workspaces ----
	p["/api/v1/workspaces"] = openapigen.PathItem{
		Post: newOp(tagWorkspaces, "createWorkspace", "Create a workspace",
			[]openapigen.Parameter{activeTenant(true)}, body("WorkspaceCreateRequest"),
			resp(sc("201", "Workspace created.", "Workspace"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagWorkspaces, "listWorkspaces", "List workspaces",
			[]openapigen.Parameter{activeTenant(false), ownerParam, qp("phase", "", openapigen.Ref("WorkspacePhase")),
				qp("q", "Keyword (matches displayName / image).", strSchema()), limitParam, continueParam},
			nil, resp(sc("200", "A page of workspaces.", "WorkspaceList"), "400", "401", "500")),
	}
	p["/api/v1/workspaces/{name}"] = openapigen.PathItem{
		Get: newOp(tagWorkspaces, "getWorkspace", "Get a workspace",
			[]openapigen.Parameter{workspaceNameP()}, nil, resp(sc("200", "Workspace detail.", "Workspace"), "401", "403", "404", "500")),
		Patch: newOp(tagWorkspaces, "updateWorkspace", "Update workspace display metadata",
			[]openapigen.Parameter{workspaceNameP()}, body("WorkspacePatchRequest"), resp(sc("200", "Updated workspace.", "Workspace"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagWorkspaces, "deleteWorkspace", "Delete a workspace",
			[]openapigen.Parameter{workspaceNameP()}, optBody("WorkspaceDeleteRequest"), resp(sc("204", "Workspace deleted.", ""), "401", "403", "404", "500")),
	}
	p["/api/v1/workspaces/{name}/start"] = openapigen.PathItem{Post: newOp(tagWorkspaces, "startWorkspace", "Start a workspace (scale to 1 replica)",
		[]openapigen.Parameter{workspaceNameP()}, nil, resp(sc("200", "Workspace started.", "Workspace"), "401", "403", "404", "409", "500"))}
	p["/api/v1/workspaces/{name}/stop"] = openapigen.PathItem{Post: newOp(tagWorkspaces, "stopWorkspace", "Stop a workspace (scale to 0 replicas)",
		[]openapigen.Parameter{workspaceNameP()}, nil, resp(sc("200", "Workspace stopped.", "Workspace"), "401", "403", "404", "409", "500"))}
	p["/api/v1/workspaces/{name}/events"] = openapigen.PathItem{Get: newOp(tagWorkspaces, "getWorkspaceEvents", "List service-level Kubernetes events for the workspace",
		[]openapigen.Parameter{workspaceNameP()}, nil, resp(sc("200", "Service-level event list.", "EventList"), "401", "403", "404", "500"))}
	p["/api/v1/workspaces/{name}/pods"] = openapigen.PathItem{Get: newOp(tagWorkspaces, "listWorkspacePods", "List Pods backing the workspace",
		[]openapigen.Parameter{workspaceNameP()}, nil, resp(sc("200", "Pod list.", "PodList"), "401", "403", "404", "500"))}
	p["/api/v1/workspaces/{name}/pods/{pod}/logs"] = openapigen.PathItem{Get: newOp(tagWorkspaces, "getWorkspacePodLogs", "Fetch or stream a workspace pod's logs",
		append([]openapigen.Parameter{workspaceNameP(), podNameP()}, logParams()...), nil, resp(logStream(), "401", "403", "404", "410", "500"))}
	p["/api/v1/workspaces/{name}/pods/{pod}/events"] = openapigen.PathItem{Get: newOp(tagWorkspaces, "listWorkspacePodEvents", "List K8s events for one pod of the workspace",
		[]openapigen.Parameter{workspaceNameP(), podNameP()}, nil, resp(sc("200", "Pod-scoped events.", "EventList"), "401", "403", "404", "500"))}

	// ---- Jobs + Runs ----
	p["/api/v1/jobs"] = openapigen.PathItem{
		Post: newOp(tagJobs, "createJob", "Create a Job (reusable template)",
			[]openapigen.Parameter{activeTenant(true)}, body("JobCreateInput"), resp(sc("201", "Job created.", "JobView"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagJobs, "listJobs", "List Jobs",
			[]openapigen.Parameter{activeTenant(false), ownerParam, qParam, limitParam, continueParam},
			nil, resp(sc("200", "A page of Jobs.", "JobList"), "400", "401", "403", "500")),
	}
	p["/api/v1/jobs/{name}"] = openapigen.PathItem{
		Get: newOp(tagJobs, "getJob", "Get a Job",
			[]openapigen.Parameter{activeTenant(true), jobNameP()}, nil, resp(sc("200", "Job detail.", "JobView"), "400", "401", "403", "404", "500")),
		Patch: newOp(tagJobs, "updateJob", "Edit a Job template / metadata",
			[]openapigen.Parameter{activeTenant(true), jobNameP()}, body("JobPatchInput"), resp(sc("200", "Job updated.", "JobView"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagJobs, "deleteJob", "Delete a Job (cascade)",
			[]openapigen.Parameter{activeTenant(true), jobNameP()}, nil, resp(sc("204", "Job (and its terminal Runs) deleted.", ""), "400", "401", "403", "404", "409", "500")),
	}
	p["/api/v1/jobs/{name}/runs"] = openapigen.PathItem{
		Post: newOp(tagJobs, "triggerRun", "Trigger a Run from a Job",
			[]openapigen.Parameter{activeTenant(true), jobNameP()}, optBody("RunTriggerInput"), resp(sc("201", "Run triggered.", "RunView"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagJobs, "listRuns", "List Runs of a Job (live)",
			[]openapigen.Parameter{activeTenant(true), jobNameP(), qp("phase", "", openapigen.Ref("RunPhase")), limitParam, continueParam},
			nil, resp(sc("200", "A page of Runs.", "RunList"), "400", "401", "403", "404", "500")),
	}
	p["/api/v1/jobs/{name}/runs/{run}"] = openapigen.PathItem{
		Get: newOp(tagJobs, "getRun", "Get a Run",
			[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, nil, resp(sc("200", "Run detail.", "RunView"), "400", "401", "403", "404", "500")),
		Delete: newOp(tagJobs, "deleteRun", "Delete a Run",
			[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, nil, resp(sc("204", "Run deleted.", ""), "400", "401", "403", "404", "500")),
	}
	p["/api/v1/jobs/{name}/runs/{run}/cancel"] = openapigen.PathItem{Post: newOp(tagJobs, "cancelRun", "Cancel a running Run",
		[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, nil, resp(sc("200", "Cancel request accepted.", "RunView"), "400", "401", "403", "404", "409", "500"))}
	p["/api/v1/jobs/{name}/runs/{run}/metrics"] = openapigen.PathItem{Get: newOp(tagJobs, "getRunMetrics", "Query resource metrics for a Run",
		append([]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, runMetricsParams()...), nil, resp(sc("200", "Metric series.", "MetricSeries"), "400", "401", "403", "404", "502", "500"))}
	p["/api/v1/jobs/{name}/runs/{run}/pods"] = openapigen.PathItem{Get: newOp(tagJobs, "listRunPods", "List Pods belonging to a Run",
		[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, nil, resp(sc("200", "Pod list.", "PodList"), "400", "401", "403", "404", "500"))}
	p["/api/v1/jobs/{name}/runs/{run}/events"] = openapigen.PathItem{Get: newOp(tagJobs, "listRunEvents", "List Run-level Kubernetes events",
		[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP()}, nil, resp(sc("200", "Job-level event list.", "EventList"), "400", "401", "403", "404", "500"))}
	p["/api/v1/jobs/{name}/runs/{run}/pods/{pod}/logs"] = openapigen.PathItem{Get: newOp(tagJobs, "getRunPodLogs", "Fetch or stream a Run pod's logs",
		append([]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP(), podNameP()}, logParams()...), nil, resp(logStream(), "400", "401", "403", "404", "410", "500"))}
	p["/api/v1/jobs/{name}/runs/{run}/pods/{pod}/events"] = openapigen.PathItem{Get: newOp(tagJobs, "listRunPodEvents", "List K8s events for one pod of a Run",
		[]openapigen.Parameter{activeTenant(true), jobNameP(), runNameP(), podNameP()}, nil, resp(sc("200", "Pod-scoped events.", "EventList"), "400", "401", "403", "404", "500"))}

	// ---- MLServices ----
	p["/api/v1/mlservices"] = openapigen.PathItem{
		Post: newOp(tagMLServices, "createMLService", "Deploy a new online inference service",
			[]openapigen.Parameter{activeTenant(true)}, body("MLServiceCreateRequest"), resp(sc("201", "Service created.", "MLService"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagMLServices, "listMLServices", "List services",
			[]openapigen.Parameter{activeTenant(false), ownerParam, qp("phase", "", openapigen.Ref("MLServicePhase")), qParam,
				qp("poolName", "Filter by resource pool.", strSchema()),
				qp("eligibleForTraffic", "Only Ready services not bound to an active traffic policy (drives traffic-policy backend pickers).", boolSchema()),
				limitParam, continueParam},
			nil, resp(sc("200", "A page of services.", "MLServiceList"), "400", "401", "500")),
	}
	p["/api/v1/mlservices/{name}"] = openapigen.PathItem{
		Get: newOp(tagMLServices, "getMLService", "Get a service",
			[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("200", "Service detail.", "MLService"), "401", "403", "404", "500")),
		Patch: newOp(tagMLServices, "updateMLService", "Update service display metadata",
			[]openapigen.Parameter{mlserviceNameP()}, body("MLServicePatchRequest"), resp(sc("200", "Updated service.", "MLService"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagMLServices, "deleteMLService", "Delete a service",
			[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("204", "Service deleted.", ""), "401", "403", "404", "500")),
	}
	p["/api/v1/mlservices/{name}/scale"] = openapigen.PathItem{Post: newOp(tagMLServices, "scaleMLService", "Set the replica count for a service",
		[]openapigen.Parameter{mlserviceNameP()}, body("MLServiceScaleRequest"), resp(sc("200", "Scale request accepted.", "MLService"), "400", "401", "403", "404", "409", "500"))}
	p["/api/v1/mlservices/{name}/start"] = openapigen.PathItem{Post: newOp(tagMLServices, "startMLService", "Start a stopped service",
		[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("200", "Start request accepted.", "MLService"), "401", "403", "404", "409", "500"))}
	p["/api/v1/mlservices/{name}/stop"] = openapigen.PathItem{Post: newOp(tagMLServices, "stopMLService", "Stop a service (scale to 0)",
		[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("200", "Stop request accepted.", "MLService"), "401", "403", "404", "409", "500"))}
	p["/api/v1/mlservices/{name}/metrics"] = openapigen.PathItem{Get: newOp(tagMLServices, "getMLServiceMetrics", "Query Prometheus for service-level metrics",
		[]openapigen.Parameter{mlserviceNameP(),
			qpReq("metric", "", openapigen.Ref("MLServiceMetricName")),
			qpReq("range", "ISO 8601 duration (5m, 15m, 1h, 6h, 24h).", strSchema()),
			qp("step", "", strSchema()),
			qp("percentile", "Only for metric=latency.", strEnum("p50", "p95", "p99")),
		}, nil, resp(sc("200", "Metric series.", "MetricSeries"), "400", "401", "403", "404", "502", "500"))}
	p["/api/v1/mlservices/{name}/pods"] = openapigen.PathItem{Get: newOp(tagMLServices, "listMLServicePods", "List Pods backing a service",
		[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("200", "Pod list.", "PodList"), "401", "403", "404", "500"))}
	p["/api/v1/mlservices/{name}/events"] = openapigen.PathItem{Get: newOp(tagMLServices, "getMLServiceEvents", "List service-level Kubernetes events",
		[]openapigen.Parameter{mlserviceNameP()}, nil, resp(sc("200", "Service-level event list.", "EventList"), "401", "403", "404", "500"))}
	p["/api/v1/mlservices/{name}/pods/{pod}/logs"] = openapigen.PathItem{Get: newOp(tagMLServices, "getMLServicePodLogs", "Fetch or stream a service pod's logs",
		append([]openapigen.Parameter{mlserviceNameP(), podNameP()}, logParams()...), nil, resp(logStream(), "401", "403", "404", "410", "500"))}
	p["/api/v1/mlservices/{name}/pods/{pod}/events"] = openapigen.PathItem{Get: newOp(tagMLServices, "listMLServicePodEvents", "List K8s events for one pod of a service",
		[]openapigen.Parameter{mlserviceNameP(), podNameP()}, nil, resp(sc("200", "Pod-scoped events.", "EventList"), "401", "403", "404", "500"))}

	// ---- Experiments + Runs ----
	p["/api/v1/experiments"] = openapigen.PathItem{
		Post: newOp(tagExperiments, "createExperiment", "Create an experiment (training template)",
			[]openapigen.Parameter{activeTenant(true)}, body("ExperimentCreateInput"), resp(sc("201", "Experiment created.", "ExperimentView"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagExperiments, "listExperiments", "List experiments",
			[]openapigen.Parameter{activeTenant(false), ownerParam, qParam, limitParam, continueParam},
			nil, resp(sc("200", "A page of experiments.", "ExperimentList"), "400", "401", "403", "500")),
	}
	p["/api/v1/experiments/{name}"] = openapigen.PathItem{
		Get: newOp(tagExperiments, "getExperiment", "Get an experiment",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, nil, resp(sc("200", "Experiment detail.", "ExperimentView"), "400", "401", "403", "404", "500")),
		Patch: newOp(tagExperiments, "updateExperiment", "Edit an experiment template / metadata",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, body("ExperimentPatchInput"), resp(sc("200", "Experiment updated.", "ExperimentView"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagExperiments, "deleteExperiment", "Delete an experiment (cascade)",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, nil, resp(sc("204", "Experiment (and its terminal Runs) deleted.", ""), "400", "401", "403", "404", "409", "500")),
	}
	p["/api/v1/experiments/{name}/runs"] = openapigen.PathItem{
		Post: newOp(tagExperiments, "triggerExperimentRun", "Trigger a Run from an experiment",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, optBody("RunTriggerInput"), resp(sc("201", "Run triggered.", "RunView"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagExperiments, "listExperimentRuns", "List Runs of an experiment (live)",
			[]openapigen.Parameter{activeTenant(true), experimentNameP(), qp("phase", "", openapigen.Ref("RunPhase")), limitParam, continueParam},
			nil, resp(sc("200", "A page of Runs.", "RunList"), "400", "401", "403", "404", "500")),
	}
	p["/api/v1/experiments/{name}/runs/{run}"] = openapigen.PathItem{
		Get: newOp(tagExperiments, "getExperimentRun", "Get a Run",
			[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, nil, resp(sc("200", "Run detail.", "RunView"), "400", "401", "403", "404", "500")),
		Delete: newOp(tagExperiments, "deleteExperimentRun", "Delete a Run",
			[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, nil, resp(sc("204", "Run deleted.", ""), "400", "401", "403", "404", "500")),
	}
	p["/api/v1/experiments/{name}/runs/{run}/cancel"] = openapigen.PathItem{Post: newOp(tagExperiments, "cancelExperimentRun", "Cancel a running Run",
		[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, nil, resp(sc("200", "Cancel request accepted.", "RunView"), "400", "401", "403", "404", "409", "500"))}
	p["/api/v1/experiments/{name}/runs/{run}/metrics"] = openapigen.PathItem{Get: newOp(tagExperiments, "getExperimentRunMetrics", "Query resource metrics for a Run",
		append([]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, runMetricsParams()...), nil, resp(sc("200", "Metric series.", "MetricSeries"), "400", "401", "403", "404", "502", "500"))}
	p["/api/v1/experiments/{name}/runs/{run}/pods"] = openapigen.PathItem{Get: newOp(tagExperiments, "listExperimentRunPods", "List Pods belonging to a Run",
		[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, nil, resp(sc("200", "Pod list.", "PodList"), "400", "401", "403", "404", "500"))}
	p["/api/v1/experiments/{name}/runs/{run}/events"] = openapigen.PathItem{Get: newOp(tagExperiments, "listExperimentRunEvents", "List Run-level Kubernetes events",
		[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP()}, nil, resp(sc("200", "Run-level event list.", "EventList"), "400", "401", "403", "404", "500"))}
	p["/api/v1/experiments/{name}/runs/{run}/pods/{pod}/logs"] = openapigen.PathItem{Get: newOp(tagExperiments, "getExperimentRunPodLogs", "Fetch or stream a Run pod's logs",
		append([]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP(), podNameP()}, logParams()...), nil, resp(logStream(), "400", "401", "403", "404", "410", "500"))}
	p["/api/v1/experiments/{name}/runs/{run}/pods/{pod}/events"] = openapigen.PathItem{Get: newOp(tagExperiments, "listExperimentRunPodEvents", "List K8s events for one pod of a Run",
		[]openapigen.Parameter{activeTenant(true), experimentNameP(), runNameP(), podNameP()}, nil, resp(sc("200", "Pod-scoped events.", "EventList"), "400", "401", "403", "404", "500"))}
	p["/api/v1/experiments/{name}/tensorboard"] = openapigen.PathItem{
		Get: newOp(tagExperiments, "getTensorBoard", "Get the experiment's TensorBoard instance",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, nil, resp(sc("200", "TensorBoard handle.", "TensorBoard"), "400", "401", "403", "404", "500")),
		Post: newOp(tagExperiments, "startTensorBoard", "Start (or reuse) a TensorBoard for the experiment",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, optBody("TensorBoardRequest"), resp(sc("200", "TensorBoard started.", "TensorBoard"), "400", "401", "403", "404", "409", "500")),
		Delete: newOp(tagExperiments, "stopTensorBoard", "Stop the experiment's TensorBoard",
			[]openapigen.Parameter{activeTenant(true), experimentNameP()}, nil, resp(sc("204", "TensorBoard stopped.", ""), "400", "401", "403", "404", "500")),
	}

	// ---- Traffic policies ----
	p["/api/v1/trafficpolicies"] = openapigen.PathItem{
		Post: newOp(tagTrafficPolicy, "createTrafficPolicy", "Create a traffic policy",
			[]openapigen.Parameter{activeTenant(true)}, body("TrafficPolicyCreateRequest"), resp(sc("201", "Traffic policy created.", "TrafficPolicy"), "400", "401", "403", "404", "409", "422", "500")),
		Get: newOp(tagTrafficPolicy, "listTrafficPolicies", "List traffic policies",
			[]openapigen.Parameter{activeTenant(false), ownerParam, qParam,
				qp("mode", "", openapigen.Ref("TrafficPolicyMode")), qp("phase", "", openapigen.Ref("TrafficPolicyPhase")),
				limitParam, continueParam},
			nil, resp(sc("200", "A page of traffic policies.", "TrafficPolicyList"), "400", "401", "500")),
	}
	p["/api/v1/trafficpolicies/{name}"] = openapigen.PathItem{
		Get: newOp(tagTrafficPolicy, "getTrafficPolicy", "Get a traffic policy",
			[]openapigen.Parameter{trafficPolicyNameP()}, nil, resp(sc("200", "Traffic policy detail.", "TrafficPolicy"), "401", "403", "404", "500")),
		Patch: newOp(tagTrafficPolicy, "updateTrafficPolicy", "Update display metadata",
			[]openapigen.Parameter{trafficPolicyNameP()}, body("TrafficPolicyPatchRequest"), resp(sc("200", "Updated traffic policy.", "TrafficPolicy"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagTrafficPolicy, "deleteTrafficPolicy", "Delete a traffic policy",
			[]openapigen.Parameter{trafficPolicyNameP()}, nil, resp(sc("204", "Traffic policy deleted.", ""), "401", "403", "404", "500")),
	}
	p["/api/v1/trafficpolicies/{name}/split"] = openapigen.PathItem{Post: newOp(tagTrafficPolicy, "splitTrafficPolicy", "Adjust traffic distribution (weights / canary percent)",
		[]openapigen.Parameter{trafficPolicyNameP()}, body("TrafficPolicySplitRequest"), resp(sc("200", "Split applied.", "TrafficPolicy"), "400", "401", "403", "404", "409", "422", "500"))}
	p["/api/v1/trafficpolicies/{name}/promote"] = openapigen.PathItem{Post: newOp(tagTrafficPolicy, "promoteTrafficPolicy", "Promote the canary backend to stable (canary only)",
		[]openapigen.Parameter{trafficPolicyNameP()}, nil, resp(sc("200", "Promoted.", "TrafficPolicy"), "400", "401", "403", "404", "409", "500"))}
	p["/api/v1/trafficpolicies/{name}/rollback"] = openapigen.PathItem{Post: newOp(tagTrafficPolicy, "rollbackTrafficPolicy", "Roll the canary back to the stable backend (canary only)",
		[]openapigen.Parameter{trafficPolicyNameP()}, nil, resp(sc("200", "Rolled back.", "TrafficPolicy"), "400", "401", "403", "404", "409", "500"))}
	p["/api/v1/trafficpolicies/{name}/metrics"] = openapigen.PathItem{Get: newOp(tagTrafficPolicy, "getTrafficPolicyMetrics", "Query per-backend metrics (grouped by backend)",
		[]openapigen.Parameter{trafficPolicyNameP(),
			qpReq("metric", "", openapigen.Ref("MLServiceMetricName")),
			qpReq("range", "ISO 8601 duration (5m, 1h, 24h).", strSchema()),
			qp("step", "", strSchema()),
			qp("backend", "Scope to one member backend (service name).", strSchema()),
		}, nil, resp(sc("200", "Metric series.", "MetricSeries"), "400", "401", "403", "404", "502", "500"))}
	p["/api/v1/trafficpolicies/{name}/events"] = openapigen.PathItem{Get: newOp(tagTrafficPolicy, "getTrafficPolicyEvents", "List traffic-policy events",
		[]openapigen.Parameter{trafficPolicyNameP()}, nil, resp(sc("200", "Event list.", "EventList"), "401", "403", "404", "500"))}

	// ---- Models / Images ----
	addArtifactKind(p, "models", tagModels, "model", "Model", "ModelStatus", "ModelInitiateRequest", "ModelInitiateResponse", "ModelCompleteRequest", "ModelList",
		[]openapigen.Parameter{})
	addArtifactKind(p, "images", tagImages, "image", "Image", "ImageStatus", "ImageInitiateRequest", "ImageInitiateResponse", "ImageCompleteRequest", "ImageList",
		[]openapigen.Parameter{qp("purpose", "Filter by definition spec.purpose.", openapigen.Ref("ImagePurpose"))})

	// ---- ResourcePools / ResourceUnits ----
	p["/api/v1/resourcepools"] = openapigen.PathItem{
		Post: newOp(tagResourcePools, "createResourcePool", "Create a resource pool",
			nil, body("ResourcePoolCreateRequest"), resp(sc("201", "Resource pool created.", "ResourcePool"), "400", "401", "403", "409", "422", "500")),
		Get: newOp(tagResourcePools, "listResourcePools", "List resource pools",
			[]openapigen.Parameter{qParam, limitParam, continueParam}, nil, resp(sc("200", "A page of pools.", "ResourcePoolList"), "401", "500")),
	}
	p["/api/v1/resourcepools/{pool}"] = openapigen.PathItem{
		Get: newOp(tagResourcePools, "getResourcePool", "Get a resource pool",
			[]openapigen.Parameter{poolNameP()}, nil, resp(sc("200", "Pool detail.", "ResourcePool"), "401", "404", "500")),
		Patch: newOp(tagResourcePools, "updateResourcePool", "Update a resource pool",
			[]openapigen.Parameter{poolNameP()}, body("ResourcePoolPatchRequest"), resp(sc("200", "Updated pool.", "ResourcePool"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagResourcePools, "deleteResourcePool", "Delete a resource pool",
			[]openapigen.Parameter{poolNameP()}, nil, resp(sc("204", "Pool deleted.", ""), "401", "403", "404", "409", "500")),
	}
	p["/api/v1/resourcepools/{pool}/units"] = openapigen.PathItem{
		Get: newOp(tagResourceUnits, "listResourceUnits", "List resource units in a pool",
			[]openapigen.Parameter{poolNameP(), limitParam, continueParam}, nil, resp(sc("200", "A page of resource units.", "ResourceUnitList"), "401", "404", "500")),
		Post: newOp(tagResourceUnits, "createResourceUnit", "Create a resource unit in a pool",
			[]openapigen.Parameter{poolNameP()}, body("ResourceUnitCreateRequest"), resp(sc("201", "Resource unit created.", "ResourceUnit"), "400", "401", "403", "404", "409", "422", "500")),
	}
	p["/api/v1/resourcepools/{pool}/units/{unit}"] = openapigen.PathItem{
		Get: newOp(tagResourceUnits, "getResourceUnit", "Get a resource unit",
			[]openapigen.Parameter{poolNameP(), unitNameP()}, nil, resp(sc("200", "Resource unit detail.", "ResourceUnit"), "401", "404", "500")),
		Patch: newOp(tagResourceUnits, "updateResourceUnit", "Update a resource unit",
			[]openapigen.Parameter{poolNameP(), unitNameP()}, body("ResourceUnitPatchRequest"), resp(sc("200", "Updated resource unit.", "ResourceUnit"), "400", "401", "403", "404", "422", "500")),
		Delete: newOp(tagResourceUnits, "deleteResourceUnit", "Delete a resource unit",
			[]openapigen.Parameter{poolNameP(), unitNameP()}, nil, resp(sc("204", "Resource unit deleted.", ""), "401", "403", "404", "409", "500")),
	}

	return p
}

// addArtifactKind registers the parallel definition + version routes shared by
// models, images, and datasets. defParams are extra list-filter query params
// specific to the kind (e.g. image purpose, dataset format).
func addArtifactKind(p map[string]openapigen.PathItem, seg, tag, kind, view, status, initReq, initResp, completeReq, list string, defParams []openapigen.Parameter) {
	K := strings.ToUpper(kind[:1]) + kind[1:]

	listParams := append([]openapigen.Parameter{activeTenant(false), qParam}, defParams...)
	listParams = append(listParams, limitParam, continueParam)
	p["/api/v1/"+seg] = openapigen.PathItem{Get: newOp(tag, "list"+K+"Definitions", "List "+kind+" definitions",
		listParams, nil, resp(sc("200", "A page of "+kind+" definitions.", "ArtifactDefinitionList"), "400", "401", "500"))}

	defPath := "/api/v1/" + seg + "/{tenant}/{name}"
	p[defPath] = openapigen.PathItem{
		Post: newOp(tag, "create"+K+"Definition", "Create a "+kind+" definition",
			[]openapigen.Parameter{tenantP(), artifactNameP()}, body("ArtifactDefinitionCreateInput"),
			resp(sc("201", K+" definition created.", "ArtifactDefinitionView"), "400", "401", "403", "409", "422", "500")),
		Get: newOp(tag, "get"+K+"Definition", "Get a "+kind+" definition",
			[]openapigen.Parameter{tenantP(), artifactNameP()}, nil,
			resp(sc("200", K+" definition detail.", "ArtifactDefinitionView"), "401", "403", "404", "500")),
		Patch: newOp(tag, "update"+K+"Definition", "Edit "+kind+" definition metadata",
			[]openapigen.Parameter{tenantP(), artifactNameP()}, body("ArtifactDefinitionPatchInput"),
			resp(sc("200", K+" definition updated.", "ArtifactDefinitionView"), "400", "401", "403", "404", "500")),
		Delete: newOp(tag, "delete"+K+"Definition", "Delete a "+kind+" definition (cascade)",
			[]openapigen.Parameter{tenantP(), artifactNameP()}, nil,
			resp(sc("204", K+" definition and its versions deleted.", ""), "401", "403", "404", "500")),
	}

	versPath := defPath + "/versions"
	p[versPath] = openapigen.PathItem{
		Get: newOp(tag, "list"+K+"Versions", "List versions of a "+kind+" definition (live)",
			[]openapigen.Parameter{tenantP(), artifactNameP(), qp("status", "", openapigen.Ref(status)), limitParam, continueParam}, nil,
			resp(sc("200", "A page of "+kind+" versions.", list), "400", "401", "404", "500")),
		Post: newOp(tag, "initiate"+K, "Initiate a "+kind+" version upload",
			[]openapigen.Parameter{tenantP(), artifactNameP()}, body(initReq),
			resp(sc("202", "Upload initiated; returns credentials and a "+kind+" handle.", initResp), "400", "401", "403", "404", "409", "422", "500")),
	}

	verPath := versPath + "/{version}"
	p[verPath] = openapigen.PathItem{
		Get: newOp(tag, "get"+K, "Get a "+kind+" artifact",
			[]openapigen.Parameter{tenantP(), artifactNameP(), artifactVerP()}, nil,
			resp(map[string]openapigen.Response{
				"200": openapigen.JSONResp(K+" detail.", view),
				"410": problemResp("Artifact has been soft-deleted; the tuple is permanently gone."),
			}, "401", "403", "404", "500")),
		Patch: newOp(tag, "update"+K, "Update mutable "+kind+" metadata",
			[]openapigen.Parameter{tenantP(), artifactNameP(), artifactVerP()}, body("ArtifactUpdateRequest"),
			resp(sc("200", K+" updated.", view), "400", "401", "403", "404", "409", "500")),
		Delete: newOp(tag, "delete"+K, "Soft-delete a "+kind+" artifact",
			[]openapigen.Parameter{tenantP(), artifactNameP(), artifactVerP()}, nil,
			resp(sc("204", K+" deleted.", ""), "401", "403", "404", "409", "500")),
	}

	p[verPath+"/complete"] = openapigen.PathItem{Post: newOp(tag, "complete"+K, "Mark a "+kind+" upload complete",
		[]openapigen.Parameter{tenantP(), artifactNameP(), artifactVerP()}, body(completeReq),
		resp(sc("200", K+" finalized.", view), "400", "401", "403", "404", "409", "422", "500"))}
	p[verPath+"/resolve"] = openapigen.PathItem{Get: newOp(tag, "resolve"+K, "Resolve download credentials for a "+kind+" version",
		[]openapigen.Parameter{tenantP(), artifactNameP(), artifactVerP()}, nil,
		resp(sc("200", "Resolved download target.", "ArtifactResolveResponse"), "401", "403", "404", "410", "500"))}
}
