//go:build integration

package integration

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// artifactHubStub is a minimal in-memory artifact-hub used to exercise the
// platform MLService create precheck (served model must be Ready). Only the
// model GET + resolve endpoints are implemented.
type artifactHubStub struct {
	mu     sync.Mutex
	models map[string]string // "ns/name/version" -> status
	engine *gin.Engine
}

func newArtifactHubStub() *artifactHubStub {
	gin.SetMode(gin.ReleaseMode)
	s := &artifactHubStub{models: map[string]string{}, engine: gin.New()}
	g := s.engine.Group("/api/v1")
	g.GET("/namespaces/:namespace/models/:name/:version", s.getModel)
	g.GET("/namespaces/:namespace/models/:name/:version/resolve", s.resolveModel)
	return s
}

func (s *artifactHubStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

func modelKey(ns, name, version string) string { return ns + "/" + name + "/" + version }

// seedModel records a model version's lifecycle status.
func (s *artifactHubStub) seedModel(ns, name, version, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[modelKey(ns, name, version)] = status
}

func (s *artifactHubStub) getModel(c *gin.Context) {
	s.mu.Lock()
	status := s.models[modelKey(c.Param("namespace"), c.Param("name"), c.Param("version"))]
	s.mu.Unlock()
	if status == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": "not-found", "title": "missing", "status": 404, "type": "x"})
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"id":        "00000000-0000-0000-0000-000000000020",
		"namespace": c.Param("namespace"),
		"name":      c.Param("name"),
		"version":   c.Param("version"),
		"kind":      "model",
		"status":    status,
		"digest":    "sha256:abc",
		"spec":      map[string]any{},
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *artifactHubStub) resolveModel(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]any{
		"uri":         "oci://registry/model:" + c.Param("version"),
		"digest":      "sha256:abc",
		"storageKind": "oci",
	})
}
