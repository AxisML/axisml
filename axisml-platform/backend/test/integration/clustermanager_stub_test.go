//go:build integration

package integration

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// clusterManagerStub is a minimal in-memory cluster-manager used by the
// integration tests. It implements just the tenant + quota endpoints the
// Tenants/Quotas slice exercises, returning the cluster-manager wire shapes the
// generated client decodes.
type clusterManagerStub struct {
	mu      sync.Mutex
	tenants map[string]map[string]any
	quotas  map[string]map[string][]map[string]any // tenant -> pool -> units
	engine  *gin.Engine
}

func newClusterManagerStub() *clusterManagerStub {
	gin.SetMode(gin.ReleaseMode)
	s := &clusterManagerStub{
		tenants: map[string]map[string]any{},
		quotas:  map[string]map[string][]map[string]any{},
		engine:  gin.New(),
	}
	g := s.engine.Group("/api/v1")
	g.POST("/tenants", s.create)
	g.GET("/tenants", s.list)
	g.GET("/tenants/:tenant", s.get)
	g.PATCH("/tenants/:tenant", s.get)
	g.DELETE("/tenants/:tenant", s.del)
	g.GET("/tenants/:tenant/quotas", s.listQuotas)
	g.POST("/tenants/:tenant/quotas", s.setQuota)
	g.PATCH("/tenants/:tenant/quotas/:pool", s.setQuota)
	g.DELETE("/tenants/:tenant/quotas/:pool", s.delQuota)
	return s
}

func (s *clusterManagerStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

// seedTenant pre-creates a Tenant CR as the System chart's seed.tenant would,
// so Platform's bootstrap can discover and import it (e.g. the built-in
// `default` tenant).
func (s *clusterManagerStub) seedTenant(name, namespace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[name] = map[string]any{"namespace": namespace}
}

func (s *clusterManagerStub) tenantBody(name string) map[string]any {
	ns, _ := s.tenants[name]["namespace"].(string)
	return map[string]any{
		"name":      name,
		"namespace": map[string]any{"name": ns},
		"quotas":    []any{},
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"phase":     "Active",
		"status":    map[string]any{"phase": "Active", "namespaceReady": true},
	}
}

func (s *clusterManagerStub) create(c *gin.Context) {
	var body struct {
		Name      string `json:"name"`
		Namespace struct {
			Name string `json:"name"`
		} `json:"namespace"`
	}
	_ = c.ShouldBindJSON(&body)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[body.Name]; ok {
		c.JSON(http.StatusConflict, gin.H{"code": "tenant-exists", "title": "exists", "status": 409, "type": "x"})
		return
	}
	s.tenants[body.Name] = map[string]any{"namespace": body.Namespace.Name}
	c.JSON(http.StatusCreated, s.tenantBody(body.Name))
}

func (s *clusterManagerStub) get(c *gin.Context) {
	name := c.Param("tenant")
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[name]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": "not-found", "title": "missing", "status": 404, "type": "x"})
		return
	}
	c.JSON(http.StatusOK, s.tenantBody(name))
}

func (s *clusterManagerStub) del(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tenants, c.Param("tenant"))
	c.Status(http.StatusNoContent)
}

func (s *clusterManagerStub) list(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]map[string]any, 0, len(s.tenants))
	for name := range s.tenants {
		items = append(items, s.tenantBody(name))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

func (s *clusterManagerStub) listQuotas(c *gin.Context) {
	tenant := c.Param("tenant")
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []map[string]any{}
	for pool, units := range s.quotas[tenant] {
		items = append(items, map[string]any{"pool": pool, "units": units})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

func (s *clusterManagerStub) setQuota(c *gin.Context) {
	tenant := c.Param("tenant")
	pool := c.Param("pool")
	var body struct {
		Pool  string           `json:"pool"`
		Units []map[string]any `json:"units"`
	}
	_ = c.ShouldBindJSON(&body)
	if pool == "" {
		pool = body.Pool
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quotas[tenant] == nil {
		s.quotas[tenant] = map[string][]map[string]any{}
	}
	s.quotas[tenant][pool] = body.Units
	c.JSON(http.StatusOK, map[string]any{"pool": pool, "units": body.Units})
}

func (s *clusterManagerStub) delQuota(c *gin.Context) {
	tenant := c.Param("tenant")
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.quotas[tenant], c.Param("pool"))
	c.Status(http.StatusNoContent)
}
