//go:build integration

package integration

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// computeServiceStub is a minimal in-memory compute-service used by the
// integration tests to exercise the platform's compute-backed behaviours: the
// metrics proxy (workload /metrics), the MLService create precheck (service
// create/get), and the tenant roll-up lists (mlruns/mlservices list, read by the
// tenant enrichCounts). It returns the compute-service wire shapes the generated
// client decodes.
type computeServiceStub struct {
	mu                  sync.Mutex
	runs                map[string][]map[string]any // ns -> mlruns
	services            map[string][]map[string]any // ns -> mlservices
	lastCreateEnv       []any                       // env of the last created service
	lastServiceTemplate map[string]any              // roles[0].template of the last created service
	lastRunTemplate     map[string]any              // roles[0].template of the last created run
	engine              *gin.Engine
}

// lastServiceEnv returns the env injected into the most recent create.
func (s *computeServiceStub) lastServiceEnv() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastCreateEnv
}

// lastServiceTmpl returns the roles[0].template of the most recent MLService
// create, so a test can assert what the platform forwarded (e.g. volumes).
func (s *computeServiceStub) lastServiceTmpl() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastServiceTemplate
}

// lastRunTmpl returns the roles[0].template of the most recent MLRun create,
// so a test can assert what the platform forwarded to compute (e.g. volumes).
func (s *computeServiceStub) lastRunTmpl() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRunTemplate
}

func newComputeServiceStub() *computeServiceStub {
	gin.SetMode(gin.ReleaseMode)
	s := &computeServiceStub{
		runs:     map[string][]map[string]any{},
		services: map[string][]map[string]any{},
		engine:   gin.New(),
	}
	g := s.engine.Group("/api/v1")
	g.GET("/namespaces/:namespace/mlruns", s.listRuns)
	g.POST("/namespaces/:namespace/mlruns", s.createRun)
	g.GET("/namespaces/:namespace/mlservices", s.listServices)
	g.POST("/namespaces/:namespace/mlservices", s.createService)
	g.GET("/namespaces/:namespace/mlservices/:name", s.getService)
	g.GET("/namespaces/:namespace/mlruns/:name/metrics", s.metrics)
	g.GET("/namespaces/:namespace/mlservices/:name/metrics", s.metrics)
	g.GET("/namespaces/:namespace/traffic-policies/:name/metrics", s.metrics)
	return s
}

// createdServices records the last created service body (name -> env) so a test
// can assert the injected model env.
func (s *computeServiceStub) serviceBody(ns, name string, env []any) map[string]any {
	return map[string]any{
		"id":        "00000000-0000-0000-0000-000000000010",
		"namespace": ns,
		"name":      name,
		"kind":      "service",
		"phase":     "Ready",
		"spec": map[string]any{
			"roles":      []any{map[string]any{"name": "default", "replicas": 1, "template": map[string]any{"image": "svc:1", "env": env}}},
			"scheduling": map[string]any{"quota": "axisml-default"},
		},
		"status":    map[string]any{},
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *computeServiceStub) createService(c *gin.Context) {
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	name, _ := body["name"].(string)
	// Echo the env from the request's single role template so a test can assert
	// the injected model URI/digest env; also record the full template so a
	// workspace test can assert the injected volumes/volumeMounts.
	var env []any
	var tmpl map[string]any
	if roles, ok := body["roles"].([]any); ok && len(roles) > 0 {
		if r0, ok := roles[0].(map[string]any); ok {
			if t, ok := r0["template"].(map[string]any); ok {
				tmpl = t
				env, _ = t["env"].([]any)
			}
		}
	}
	s.mu.Lock()
	s.lastCreateEnv = env
	s.lastServiceTemplate = tmpl
	s.mu.Unlock()
	c.JSON(http.StatusCreated, s.serviceBody(c.Param("namespace"), name, env))
}

func (s *computeServiceStub) getService(c *gin.Context) {
	c.JSON(http.StatusOK, s.serviceBody(c.Param("namespace"), c.Param("name"), nil))
}

// createRun records the incoming role template (so a test can assert forwarded
// fields such as volumes/volumeMounts) and echoes back a minimal MLRun the
// generated client can decode.
func (s *computeServiceStub) createRun(c *gin.Context) {
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	name, _ := body["name"].(string)
	var tmpl map[string]any
	if roles, ok := body["roles"].([]any); ok && len(roles) > 0 {
		if r0, ok := roles[0].(map[string]any); ok {
			tmpl, _ = r0["template"].(map[string]any)
		}
	}
	s.mu.Lock()
	s.lastRunTemplate = tmpl
	s.mu.Unlock()
	ns := c.Param("namespace")
	c.JSON(http.StatusCreated, map[string]any{
		"id":        "00000000-0000-0000-0000-000000000020",
		"namespace": ns,
		"name":      name,
		"phase":     "Pending",
		"spec":      map[string]any{"scheduling": map[string]any{"quota": "axisml-default"}},
		"status":    map[string]any{},
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *computeServiceStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

func (s *computeServiceStub) listRuns(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.runs[c.Param("namespace")]
	if items == nil {
		items = []map[string]any{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items), "total": len(items), "continueToken": ""})
}

func (s *computeServiceStub) listServices(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.services[c.Param("namespace")]
	if items == nil {
		items = []map[string]any{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items), "total": len(items), "continueToken": ""})
}

func (s *computeServiceStub) metrics(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]any{
		"metric": c.Query("metric"),
		"range":  c.Query("range"),
		"unit":   "cores",
		"series": []map[string]any{{"timestamp": time.Now().UTC().Format(time.RFC3339), "value": 1.5}},
	})
}
