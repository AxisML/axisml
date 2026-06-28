// Package volume hosts the REST handlers for durable data-volume materialisation.
// Each Volume is backed by a namespace-scoped PersistentVolumeClaim (Kubernetes)
// or a managed Docker volume (Lite). All handlers are stateless and translate
// HTTP requests into VolumeManager calls. cluster-manager does not interpret the
// volume's purpose — naming and mounting are the caller's job (design §4.5).
package volume

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	srv "github.com/axisml/axisml/components/cluster-manager/internal/server"
	"github.com/axisml/axisml/components/cluster-manager/pkg/extensions"
)

// Handler implements the /api/v1/volumes[/{namespace}/{name}] HTTP surface. It
// owns no state; all access goes through the injected VolumeManager (Kubernetes
// PVC or Lite Docker volume).
type Handler struct {
	volumes extensions.VolumeManager
}

// NewHandler builds a volume handler over the given store.
func NewHandler(volumes extensions.VolumeManager) *Handler {
	return &Handler{volumes: volumes}
}

// Register attaches the volume routes to the provided /api/v1 group.
func (h *Handler) Register(rg *gin.RouterGroup) {
	v := rg.Group("/volumes")
	v.POST("", h.Create)
	v.GET("", h.List)
	v.GET("/:namespace/:name", h.Get)
	v.PATCH("/:namespace/:name", h.Patch)
	v.DELETE("/:namespace/:name", h.Delete)
	rg.GET("/storageclasses", h.ListStorageClasses)
}

// ListStorageClasses handles GET /api/v1/storageclasses — the catalog of storage
// backends available when creating a volume.
func (h *Handler) ListStorageClasses(c *gin.Context) {
	classes, err := h.volumes.ListStorageClasses(c.Request.Context())
	if err != nil {
		writeK8sError(c, err, "")
		return
	}
	items := srv.StorageClassesToAPI(classes)
	c.JSON(http.StatusOK, srv.StorageClassList{Items: items, Count: len(items)})
}

// Create handles POST /api/v1/volumes. Idempotent: an existing volume is
// treated as success so a caller retry is safe.
func (h *Handler) Create(c *gin.Context) {
	var req srv.CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if req.Namespace == "" || req.Name == "" {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidVolume",
			"namespace and name are required", "")
		return
	}
	if req.Size == "" {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidSize", "size is required", "")
		return
	}

	pvc, err := srv.APIToPVC(req)
	if err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidSize",
			"size is not a valid Kubernetes Quantity", err.Error())
		return
	}
	if err := h.volumes.Ensure(c.Request.Context(), pvc); err != nil {
		writeK8sError(c, err, req.Namespace+"/"+req.Name)
		return
	}
	c.JSON(http.StatusCreated, srv.VolumeToAPI(pvc))
}

// List handles GET /api/v1/volumes?namespace=&labelSelector=. Returns the managed
// volumes (with live phase/capacity, without per-item mounts).
func (h *Handler) List(c *gin.Context) {
	pvcs, err := h.volumes.List(c.Request.Context(), c.Query("namespace"), c.Query("labelSelector"))
	if err != nil {
		writeK8sError(c, err, "")
		return
	}
	resp := srv.VolumeList{Items: make([]srv.Volume, 0, len(pvcs))}
	for i := range pvcs {
		resp.Items = append(resp.Items, srv.VolumeToAPI(&pvcs[i]))
	}
	resp.Count = len(resp.Items)
	c.JSON(http.StatusOK, resp)
}

// Get handles GET /api/v1/volumes/{namespace}/{name}. Includes the mount
// occupancy in the status.
func (h *Handler) Get(c *gin.Context) {
	key := types.NamespacedName{Namespace: c.Param("namespace"), Name: c.Param("name")}
	pvc, err := h.volumes.Get(c.Request.Context(), key)
	if err != nil {
		writeK8sError(c, err, key.String())
		return
	}
	v := srv.VolumeToAPI(pvc)
	if mounts, err := h.volumes.Mounts(c.Request.Context(), key); err == nil {
		if v.Status == nil {
			v.Status = &srv.VolumeStatus{}
		}
		v.Status.Mounts = srv.MountsToAPI(mounts)
	}
	c.JSON(http.StatusOK, v)
}

// Patch handles PATCH /api/v1/volumes/{namespace}/{name}: expand size and/or
// update description / labels. storageClass and accessModes are immutable.
func (h *Handler) Patch(c *gin.Context) {
	key := types.NamespacedName{Namespace: c.Param("namespace"), Name: c.Param("name")}
	var req srv.PatchVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	pvc, err := h.volumes.Patch(c.Request.Context(), key, req.VolumePatch())
	if err != nil {
		writeK8sError(c, err, key.String())
		return
	}
	c.JSON(http.StatusOK, srv.VolumeToAPI(pvc))
}

// Delete handles DELETE /api/v1/volumes/{namespace}/{name}. Occupancy-guarded:
// a volume mounted by a running pod is refused with 409 unless ?force=true. A
// missing volume is success (idempotent 204).
func (h *Handler) Delete(c *gin.Context) {
	key := types.NamespacedName{Namespace: c.Param("namespace"), Name: c.Param("name")}
	if c.Query("force") != "true" {
		// Fail closed: if occupancy can't be determined (e.g. a transient pod
		// LIST failure), refuse the delete rather than risk reclaiming a volume
		// a running workload still mounts.
		mounts, err := h.volumes.Mounts(c.Request.Context(), key)
		if err != nil {
			writeK8sError(c, err, key.String())
			return
		}
		for _, m := range mounts {
			if m.Running {
				srv.AbortWithProblem(c, http.StatusConflict, "VolumeInUse",
					"volume is mounted by running workloads; stop them or use force=true", key.String())
				return
			}
		}
	}
	if err := h.volumes.Delete(c.Request.Context(), key); err != nil {
		if apierrors.IsNotFound(err) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, key.String())
		return
	}
	c.Status(http.StatusNoContent)
}

func writeK8sError(c *gin.Context, err error, name string) {
	switch {
	case errors.Is(err, extensions.ErrCapabilityUnavailable):
		srv.AbortWithProblem(c, http.StatusConflict, "CapabilityUnavailable",
			"operation not supported in this deployment form", name)
	case apierrors.IsNotFound(err):
		srv.AbortWithProblem(c, http.StatusNotFound, "NotFound", "resource not found", name)
	case apierrors.IsInvalid(err):
		srv.AbortWithProblem(c, http.StatusUnprocessableEntity, "Invalid",
			"K8s API rejected the volume", err.Error())
	case apierrors.IsForbidden(err):
		srv.AbortWithProblem(c, http.StatusUnprocessableEntity, "Forbidden",
			"K8s API forbade the change (e.g. volume shrink)", err.Error())
	case apierrors.IsBadRequest(err):
		srv.AbortWithProblem(c, http.StatusBadRequest, "BadRequest", err.Error(), "")
	default:
		srv.AbortWithProblem(c, http.StatusInternalServerError, "K8sError",
			"unexpected error from Kubernetes API", err.Error())
	}
}
