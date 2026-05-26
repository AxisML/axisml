// Package resourcepool hosts the REST handlers for the ResourcePool CRD and
// its embedded `spec.units[]` array. All handlers are stateless and
// translate HTTP requests into K8s API calls on axisml.io/v1alpha1.ResourcePool.
package resourcepool

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	axismlv1alpha1 "github.com/axisml/axisml/components/cluster-manager/api/v1alpha1"
	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
)

// Handler implements the /api/v1/resource-pools[/{pool}[/resource-units...]]
// HTTP surface. It owns no state; all mutations go straight to the K8s API
// server via client.Client.
type Handler struct {
	Client client.Client
}

// Register attaches all routes to the provided /api/v1 group.
func (h *Handler) Register(rg *gin.RouterGroup) {
	pools := rg.Group("/resource-pools")
	pools.POST("", h.Create)
	pools.GET("", h.List)
	pools.GET("/:pool", h.Get)
	pools.PATCH("/:pool", h.Patch)
	pools.DELETE("/:pool", h.Delete)

	units := pools.Group("/:pool/resource-units")
	units.POST("", h.CreateUnit)
	units.GET("", h.ListUnits)
	units.GET("/:unit", h.GetUnit)
	units.PATCH("/:unit", h.PatchUnit)
	units.DELETE("/:unit", h.DeleteUnit)
}

// ─────────────────────────────────────────────────────────────── Pools

// Create handles POST /api/v1/resource-pools.
func (h *Handler) Create(c *gin.Context) {
	var req srv.CreateResourcePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := srv.ValidateDNS1123Name("name", req.Name); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidName", err.Error(), "")
		return
	}
	if dup, name := firstDuplicateUnit(req.Units); dup {
		srv.AbortWithProblem(c, http.StatusBadRequest, "DuplicateUnit",
			"unit names must be unique within a pool", "duplicate: "+name)
		return
	}
	for _, u := range req.Units {
		if err := srv.ValidateDNS1123Name("units["+u.Name+"].name", u.Name); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidUnitName", err.Error(), "")
			return
		}
	}

	pool := srv.DTOToPool(req, c.GetHeader(srv.HeaderUser))
	if err := h.Client.Create(c.Request.Context(), pool); err != nil {
		writeK8sError(c, err, req.Name)
		return
	}
	c.JSON(http.StatusCreated, srv.PoolToDTO(pool))
}

// Get handles GET /api/v1/resource-pools/{pool}.
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("pool")
	pool, err := h.getPool(c.Request.Context(), name)
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.PoolToDTO(pool))
}

// List handles GET /api/v1/resource-pools.
func (h *Handler) List(c *gin.Context) {
	opts := []client.ListOption{}
	if sel := c.Query("labelSelector"); sel != "" {
		ps, err := labelSelectorFrom(sel)
		if err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidSelector",
				"labelSelector parse error", err.Error())
			return
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: ps})
	}

	var pools axismlv1alpha1.ResourcePoolList
	if err := h.Client.List(c.Request.Context(), &pools, opts...); err != nil {
		writeK8sError(c, err, "")
		return
	}

	resp := srv.ResourcePoolList{Items: make([]srv.ResourcePoolDTO, 0, len(pools.Items))}
	for i := range pools.Items {
		resp.Items = append(resp.Items, srv.PoolToDTO(&pools.Items[i]))
	}
	resp.Count = len(resp.Items)
	resp.ContinueToken = pools.Continue
	c.JSON(http.StatusOK, resp)
}

// Patch handles PATCH /api/v1/resource-pools/{pool}.
func (h *Handler) Patch(c *gin.Context) {
	name := c.Param("pool")
	var req srv.PatchResourcePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}

	pool, err := h.getPool(c.Request.Context(), name)
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	base := pool.DeepCopy()

	if req.NodeSelector != nil {
		pool.Spec.NodeSelector = req.NodeSelector
	}
	if req.Tolerations != nil {
		pool.Spec.Tolerations = req.Tolerations
	}
	if req.Labels != nil {
		pool.Labels = req.Labels
	}
	if pool.Annotations == nil {
		pool.Annotations = map[string]string{}
	}
	if req.Annotations != nil {
		// User-supplied annotations replace user-visible ones; preserve
		// reserved (description, last-modified-by) below.
		desc := pool.Annotations[srv.DescriptionAnnotation]
		lmb := pool.Annotations[srv.LastModifiedByAnnotation]
		pool.Annotations = map[string]string{}
		for k, v := range req.Annotations {
			pool.Annotations[k] = v
		}
		if desc != "" {
			pool.Annotations[srv.DescriptionAnnotation] = desc
		}
		if lmb != "" {
			pool.Annotations[srv.LastModifiedByAnnotation] = lmb
		}
	}
	if req.Description != nil {
		pool.Annotations[srv.DescriptionAnnotation] = *req.Description
	}
	if user := c.GetHeader(srv.HeaderUser); user != "" {
		pool.Annotations[srv.LastModifiedByAnnotation] = user
	}

	if err := h.Client.Patch(c.Request.Context(), pool, client.MergeFrom(base)); err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.PoolToDTO(pool))
}

// Delete handles DELETE /api/v1/resource-pools/{pool}.
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("pool")
	pool := &axismlv1alpha1.ResourcePool{}
	pool.Name = name
	if err := h.Client.Delete(c.Request.Context(), pool); err != nil {
		if apierrors.IsNotFound(err) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, name)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────── Units

// CreateUnit handles POST .../resource-units. Optimistically patches the
// parent ResourcePool's spec.units[].
func (h *Handler) CreateUnit(c *gin.Context) {
	poolName := c.Param("pool")
	var req srv.CreateResourceUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := srv.ValidateDNS1123Name("name", req.Name); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidName", err.Error(), "")
		return
	}

	pool, err := h.getPool(c.Request.Context(), poolName)
	if err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	for _, u := range pool.Spec.Units {
		if u.Name == req.Name {
			srv.AbortWithProblem(c, http.StatusConflict, "DuplicateUnit",
				"unit name already exists in pool", req.Name)
			return
		}
	}

	base := pool.DeepCopy()
	pool.Spec.Units = append(pool.Spec.Units, srv.DTOToUnit(req))
	if pool.Annotations == nil {
		pool.Annotations = map[string]string{}
	}
	if user := c.GetHeader(srv.HeaderUser); user != "" {
		pool.Annotations[srv.LastModifiedByAnnotation] = user
	}
	if err := h.Client.Patch(c.Request.Context(), pool, client.MergeFrom(base)); err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	c.JSON(http.StatusCreated, srv.UnitToDTO(pool.Spec.Units[len(pool.Spec.Units)-1]))
}

// ListUnits handles GET .../resource-units. Returns pool.spec.units[].
func (h *Handler) ListUnits(c *gin.Context) {
	pool, err := h.getPool(c.Request.Context(), c.Param("pool"))
	if err != nil {
		writeK8sError(c, err, c.Param("pool"))
		return
	}
	items := make([]srv.ResourceUnitDTO, 0, len(pool.Spec.Units))
	for _, u := range pool.Spec.Units {
		items = append(items, srv.UnitToDTO(u))
	}
	c.JSON(http.StatusOK, srv.ResourceUnitList{Items: items, Count: len(items)})
}

// GetUnit handles GET .../resource-units/{unit}.
func (h *Handler) GetUnit(c *gin.Context) {
	pool, err := h.getPool(c.Request.Context(), c.Param("pool"))
	if err != nil {
		writeK8sError(c, err, c.Param("pool"))
		return
	}
	for _, u := range pool.Spec.Units {
		if u.Name == c.Param("unit") {
			c.JSON(http.StatusOK, srv.UnitToDTO(u))
			return
		}
	}
	srv.AbortWithProblem(c, http.StatusNotFound, "UnitNotFound",
		"unit not found in pool", c.Param("unit"))
}

// PatchUnit handles PATCH .../resource-units/{unit}.
func (h *Handler) PatchUnit(c *gin.Context) {
	poolName, unitName := c.Param("pool"), c.Param("unit")
	var req srv.PatchResourceUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}

	pool, err := h.getPool(c.Request.Context(), poolName)
	if err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	idx := -1
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == unitName {
			idx = i
			break
		}
	}
	if idx == -1 {
		srv.AbortWithProblem(c, http.StatusNotFound, "UnitNotFound",
			"unit not found in pool", unitName)
		return
	}

	base := pool.DeepCopy()
	u := &pool.Spec.Units[idx]
	if req.Requests != nil {
		u.Requests = req.Requests
	}
	if req.Limits != nil {
		u.Limits = req.Limits
	}
	if req.NodeSelector != nil {
		u.NodeSelector = req.NodeSelector
	}
	if req.Annotations != nil {
		desc := ""
		if u.Annotations != nil {
			desc = u.Annotations[srv.DescriptionAnnotation]
		}
		u.Annotations = map[string]string{}
		for k, v := range req.Annotations {
			u.Annotations[k] = v
		}
		if desc != "" {
			u.Annotations[srv.DescriptionAnnotation] = desc
		}
	}
	if req.Description != nil {
		if u.Annotations == nil {
			u.Annotations = map[string]string{}
		}
		u.Annotations[srv.DescriptionAnnotation] = *req.Description
	}
	if user := c.GetHeader(srv.HeaderUser); user != "" {
		if pool.Annotations == nil {
			pool.Annotations = map[string]string{}
		}
		pool.Annotations[srv.LastModifiedByAnnotation] = user
	}
	if err := h.Client.Patch(c.Request.Context(), pool, client.MergeFrom(base)); err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	c.JSON(http.StatusOK, srv.UnitToDTO(pool.Spec.Units[idx]))
}

// DeleteUnit handles DELETE .../resource-units/{unit}.
func (h *Handler) DeleteUnit(c *gin.Context) {
	poolName, unitName := c.Param("pool"), c.Param("unit")
	pool, err := h.getPool(c.Request.Context(), poolName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, poolName)
		return
	}
	idx := -1
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == unitName {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.Status(http.StatusNoContent)
		return
	}
	base := pool.DeepCopy()
	pool.Spec.Units = append(pool.Spec.Units[:idx], pool.Spec.Units[idx+1:]...)
	if user := c.GetHeader(srv.HeaderUser); user != "" {
		if pool.Annotations == nil {
			pool.Annotations = map[string]string{}
		}
		pool.Annotations[srv.LastModifiedByAnnotation] = user
	}
	if err := h.Client.Patch(c.Request.Context(), pool, client.MergeFrom(base)); err != nil {
		writeK8sError(c, err, poolName)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────── helpers

func (h *Handler) getPool(ctx context.Context, name string) (*axismlv1alpha1.ResourcePool, error) {
	pool := &axismlv1alpha1.ResourcePool{}
	if err := h.Client.Get(ctx, types.NamespacedName{Name: name}, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func writeK8sError(c *gin.Context, err error, name string) {
	switch {
	case apierrors.IsAlreadyExists(err):
		srv.AbortWithProblem(c, http.StatusConflict, "AlreadyExists",
			"resource already exists", name)
	case apierrors.IsNotFound(err):
		srv.AbortWithProblem(c, http.StatusNotFound, "NotFound",
			"resource not found", name)
	case apierrors.IsConflict(err):
		srv.AbortWithProblem(c, http.StatusConflict, "OptimisticLockConflict",
			"resource was modified concurrently — retry", name)
	case apierrors.IsInvalid(err):
		srv.AbortWithProblem(c, http.StatusUnprocessableEntity, "Invalid",
			"K8s API rejected the resource", err.Error())
	case apierrors.IsBadRequest(err):
		srv.AbortWithProblem(c, http.StatusBadRequest, "BadRequest", err.Error(), "")
	default:
		// Map K8s "no kind X in scheme" / generic errors to 500.
		srv.AbortWithProblem(c, http.StatusInternalServerError, "K8sError",
			"unexpected error from Kubernetes API", err.Error())
	}
	_ = errors.New
}

func firstDuplicateUnit(units []srv.CreateResourceUnitRequest) (bool, string) {
	seen := map[string]struct{}{}
	for _, u := range units {
		if _, ok := seen[u.Name]; ok {
			return true, u.Name
		}
		seen[u.Name] = struct{}{}
	}
	return false, ""
}
