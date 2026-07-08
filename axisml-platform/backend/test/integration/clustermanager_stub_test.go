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
	quotas  map[string]map[string]map[string]any // tenant -> pool -> quota object ({pool,units} or {pool,quota})
	volumes map[string]map[string]any            // "namespace/name" -> volume
	engine  *gin.Engine
}

func newClusterManagerStub() *clusterManagerStub {
	gin.SetMode(gin.ReleaseMode)
	s := &clusterManagerStub{
		tenants: map[string]map[string]any{},
		quotas:  map[string]map[string]map[string]any{},
		volumes: map[string]map[string]any{},
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
	g.POST("/volumes", s.createVolume)
	g.GET("/volumes", s.listVolumes)
	g.GET("/volumes/:namespace/:name", s.getVolume)
	g.PATCH("/volumes/:namespace/:name", s.patchVolume)
	g.DELETE("/volumes/:namespace/:name", s.delVolume)
	g.GET("/resourcepools/:pool/usage", s.poolUsage)
	g.GET("/resourcepools/:pool/metrics", s.poolMetrics)
	return s
}

// seedQuota records a tenant's quota in a pool so listQuotas (and thus the
// dashboard's per-pool fold) returns it.
func (s *clusterManagerStub) seedQuota(tenant, pool string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quotas[tenant] == nil {
		s.quotas[tenant] = map[string]map[string]any{}
	}
	s.quotas[tenant][pool] = map[string]any{
		"pool":  pool,
		"units": []map[string]any{{"unitName": "small", "quantity": 1}},
	}
}

func (s *clusterManagerStub) poolUsage(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]any{
		"pool":   c.Param("pool"),
		"tenant": c.Query("tenant"),
		"meters": []map[string]any{
			{"resource": "cpu", "used": 2.0, "total": 8.0, "unit": "cores"},
			{"resource": "nvidia.com/gpu", "used": 1.0, "total": 4.0, "unit": "cards"},
		},
	})
}

func (s *clusterManagerStub) poolMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]any{
		"metric": c.Query("metric"),
		"range":  c.Query("range"),
		"unit":   "cores",
		"series": []map[string]any{{"timestamp": time.Now().UTC().Format(time.RFC3339), "value": 1.5}},
	})
}

func volKey(ns, name string) string { return ns + "/" + name }

func (s *clusterManagerStub) createVolume(c *gin.Context) {
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	ns, _ := body["namespace"].(string)
	name, _ := body["name"].(string)
	s.mu.Lock()
	defer s.mu.Unlock()
	v := map[string]any{
		"namespace":    ns,
		"name":         name,
		"size":         body["size"],
		"storageClass": body["storageClass"],
		"accessModes":  body["accessModes"],
		"description":  body["description"],
		"labels":       body["labels"],
		"status":       map[string]any{"phase": "Bound", "boundCapacity": body["size"]},
		"createdAt":    time.Now().UTC().Format(time.RFC3339),
	}
	s.volumes[volKey(ns, name)] = v
	c.JSON(http.StatusCreated, v)
}

func (s *clusterManagerStub) listVolumes(c *gin.Context) {
	ns := c.Query("namespace")
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []map[string]any{}
	for _, v := range s.volumes {
		if ns == "" || v["namespace"] == ns {
			items = append(items, v)
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

func (s *clusterManagerStub) getVolume(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.volumes[volKey(c.Param("namespace"), c.Param("name"))]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": "not-found", "title": "missing", "status": 404, "type": "x"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (s *clusterManagerStub) patchVolume(c *gin.Context) {
	var body map[string]any
	_ = c.ShouldBindJSON(&body)
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.volumes[volKey(c.Param("namespace"), c.Param("name"))]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"code": "not-found", "title": "missing", "status": 404, "type": "x"})
		return
	}
	for _, k := range []string{"size", "description", "labels"} {
		if val, present := body[k]; present {
			v[k] = val
		}
	}
	c.JSON(http.StatusOK, v)
}

func (s *clusterManagerStub) delVolume(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.volumes, volKey(c.Param("namespace"), c.Param("name")))
	c.Status(http.StatusNoContent)
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
	for _, q := range s.quotas[tenant] {
		items = append(items, q)
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

func (s *clusterManagerStub) setQuota(c *gin.Context) {
	tenant := c.Param("tenant")
	pool := c.Param("pool")
	var body struct {
		Pool  string           `json:"pool"`
		Units []map[string]any `json:"units"`
		Quota map[string]any   `json:"quota"`
	}
	_ = c.ShouldBindJSON(&body)
	if pool == "" {
		pool = body.Pool
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quotas[tenant] == nil {
		s.quotas[tenant] = map[string]map[string]any{}
	}
	// Round-trip whichever mode the caller sent (units or direct min/max).
	entry := map[string]any{"pool": pool}
	if body.Quota != nil {
		entry["quota"] = body.Quota
	} else {
		entry["units"] = body.Units
	}
	s.quotas[tenant][pool] = entry
	c.JSON(http.StatusOK, entry)
}

func (s *clusterManagerStub) delQuota(c *gin.Context) {
	tenant := c.Param("tenant")
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.quotas[tenant], c.Param("pool"))
	c.Status(http.StatusNoContent)
}
