// Package volume hosts the REST handlers for durable Volume materialisation.
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
// owns no state; all writes go through the injected VolumeManager (Kubernetes PVC
// or Lite Docker volume).
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
	v.DELETE("/:namespace/:name", h.Delete)
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

// Delete handles DELETE /api/v1/volumes/{namespace}/{name}. A missing volume is
// success (idempotent 204).
func (h *Handler) Delete(c *gin.Context) {
	key := types.NamespacedName{Namespace: c.Param("namespace"), Name: c.Param("name")}
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
	case apierrors.IsBadRequest(err):
		srv.AbortWithProblem(c, http.StatusBadRequest, "BadRequest", err.Error(), "")
	default:
		srv.AbortWithProblem(c, http.StatusInternalServerError, "K8sError",
			"unexpected error from Kubernetes API", err.Error())
	}
}
