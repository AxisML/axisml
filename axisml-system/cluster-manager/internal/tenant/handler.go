// Package tenant hosts the REST handlers for the cluster-scoped Tenant CR
// (axisml.io/v1alpha1, owned by tenant-operator's API types). cluster-manager
// is the REST writer of the Tenant CR spec; tenant-operator reconciles it into
// a Namespace + ElasticQuota + per-tenant init resources and writes status.
//
// Handlers are stateless: every mutation is a direct K8s API call (with a
// single optimistic-lock retry). Quotas are accepted either in the business form
// (`unit × quantity` per pool) or as direct min/max resources. Both forms are
// compiled into Tenant.spec.quotas[].min/max before being written to the CR.
package tenant

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/util/retry"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	srv "github.com/axisml/axisml/axisml-system/cluster-manager/internal/server"
	"github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
)

// Handler implements /api/v1/tenants[/{tenant}[/quotas...]]. It owns no state;
// reads/writes go through the injected stores (Kubernetes CRD or static config).
// Quota folding reads the ResourcePool store.
type Handler struct {
	tenants extensions.TenantProvider
	pools   extensions.ResourcePoolProvider
}

// NewHandler builds a tenant handler over the given stores.
func NewHandler(tenants extensions.TenantProvider, pools extensions.ResourcePoolProvider) *Handler {
	return &Handler{tenants: tenants, pools: pools}
}

// Register attaches all routes to the provided /api/v1 group.
func (h *Handler) Register(rg *gin.RouterGroup) {
	tenants := rg.Group("/tenants")
	tenants.POST("", h.Create)
	tenants.GET("", h.List)
	tenants.GET("/:tenant", h.Get)
	tenants.PATCH("/:tenant", h.Patch)
	tenants.DELETE("/:tenant", h.Delete)

	quotas := tenants.Group("/:tenant/quotas")
	quotas.GET("", h.ListQuotas)
	quotas.POST("", h.SetQuota)
	quotas.PATCH("/:pool", h.PatchQuota)
	quotas.DELETE("/:pool", h.DeleteQuota)
}

// ─────────────────────────────────────────────────────────────── Tenants

// Create handles POST /api/v1/tenants.
func (h *Handler) Create(c *gin.Context) {
	var req srv.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if err := srv.ValidateDNS1123Name("name", req.Name); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidName", err.Error(), "")
		return
	}

	folded, anno, err := h.foldQuotas(c.Request.Context(), req.Quotas)
	if err != nil {
		h.writeQuotaError(c, err)
		return
	}
	cr := srv.APIToTenant(req, folded, anno, c.GetHeader(srv.HeaderUser))
	// tenant-operator requires a non-empty tenant-id label (its orphan-detection
	// anchor) before it will provision the namespace. Stamp a stable UUID when
	// the caller didn't supply one so every REST-created tenant provisions like
	// the Helm-seeded one.
	if cr.Labels == nil {
		cr.Labels = map[string]string{}
	}
	if cr.Labels[tenantv1alpha1.LabelTenantID] == "" {
		cr.Labels[tenantv1alpha1.LabelTenantID] = string(uuid.NewUUID())
	}
	if err := h.tenants.Create(c.Request.Context(), cr); err != nil {
		writeK8sError(c, err, req.Name)
		return
	}
	c.JSON(http.StatusCreated, srv.TenantToAPI(cr))
}

// Get handles GET /api/v1/tenants/{tenant}.
func (h *Handler) Get(c *gin.Context) {
	name := c.Param("tenant")
	cr, err := h.getTenant(c.Request.Context(), name)
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.TenantToAPI(cr))
}

// List handles GET /api/v1/tenants. Supports ?labelSelector, ?limit (1-500,
// default server-side), and ?continue (opaque cursor).
func (h *Handler) List(c *gin.Context) {
	opts := metav1.ListOptions{}
	if sel := c.Query("labelSelector"); sel != "" {
		if _, err := labels.Parse(sel); err != nil {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidSelector", "labelSelector parse error", err.Error())
			return
		}
		opts.LabelSelector = sel
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidLimit", "limit must be an integer in [1, 500]", v)
			return
		}
		opts.Limit = int64(n)
	}
	opts.Continue = c.Query("continue")

	list, err := h.tenants.List(c.Request.Context(), opts)
	if err != nil {
		writeK8sError(c, err, "")
		return
	}
	resp := srv.TenantList{Items: make([]srv.Tenant, 0, len(list.Items))}
	for i := range list.Items {
		resp.Items = append(resp.Items, srv.TenantToAPI(&list.Items[i]))
	}
	resp.Count = len(resp.Items)
	resp.ContinueToken = list.Continue
	c.JSON(http.StatusOK, resp)
}

// Patch handles PATCH /api/v1/tenants/{tenant}. Updates namespace labels/
// annotations, initResources, and CR labels/annotations. `name` and
// `spec.namespace.name` are immutable; quotas mutate via the sub-routes.
func (h *Handler) Patch(c *gin.Context) {
	name := c.Param("tenant")
	var req srv.PatchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}

	user := c.GetHeader(srv.HeaderUser)
	var result *tenantv1alpha1.Tenant
	err := h.mutateWithRetry(c.Request.Context(), name, func(t *tenantv1alpha1.Tenant) error {
		applyTenantPatch(t, req, user)
		result = t
		return nil
	})
	if err != nil {
		writeK8sError(c, err, name)
		return
	}
	c.JSON(http.StatusOK, srv.TenantToAPI(result))
}

// applyTenantPatch mutates the Tenant CR in place per the PATCH request. Kept
// as a closure body so mutateWithRetry can replay it against a fresh read on a
// 409 conflict.
func applyTenantPatch(t *tenantv1alpha1.Tenant, req srv.PatchTenantRequest, user string) {
	if req.NamespaceLabels != nil {
		t.Spec.Namespace.Labels = req.NamespaceLabels
	}
	if req.NamespaceAnnotations != nil {
		t.Spec.Namespace.Annotations = req.NamespaceAnnotations
	}
	if req.InitResources != nil {
		t.Spec.InitResources = *req.InitResources
	}
	if req.Labels != nil {
		t.Labels = req.Labels
	}
	if req.Annotations != nil {
		// User-supplied annotations replace user-visible ones; preserve the
		// reserved last-modified-by / quotas round-trip annotations below.
		lmb := t.Annotations[srv.LastModifiedByAnnotation]
		quotas := t.Annotations[srv.QuotasAnnotation]
		t.Annotations = map[string]string{}
		for k, v := range req.Annotations {
			t.Annotations[k] = v
		}
		if lmb != "" {
			t.Annotations[srv.LastModifiedByAnnotation] = lmb
		}
		if quotas != "" {
			t.Annotations[srv.QuotasAnnotation] = quotas
		}
	}
	if user != "" {
		if t.Annotations == nil {
			t.Annotations = map[string]string{}
		}
		t.Annotations[srv.LastModifiedByAnnotation] = user
	}
}

// Delete handles DELETE /api/v1/tenants/{tenant}. Hard-deletes the CR; the
// durable tenant record and soft-delete/retention live in Platform.
func (h *Handler) Delete(c *gin.Context) {
	name := c.Param("tenant")
	if err := h.tenants.Delete(c.Request.Context(), name); err != nil {
		if apierrors.IsNotFound(err) {
			c.Status(http.StatusNoContent)
			return
		}
		writeK8sError(c, err, name)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────── Quotas

// ListQuotas handles GET .../quotas, returning the business-form selection.
func (h *Handler) ListQuotas(c *gin.Context) {
	cr, err := h.getTenant(c.Request.Context(), c.Param("tenant"))
	if err != nil {
		writeK8sError(c, err, c.Param("tenant"))
		return
	}
	items := srv.TenantToAPI(cr).Quotas
	c.JSON(http.StatusOK, srv.QuotaList{Items: items, Count: len(items)})
}

// SetQuota handles POST .../quotas — create or replace one pool's quota.
func (h *Handler) SetQuota(c *gin.Context) {
	tenant := c.Param("tenant")
	var req srv.SetQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	if req.Pool == "" {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidQuota", "pool is required", "")
		return
	}
	q := srv.Quota(req)
	var result srv.Quota
	err := h.mutateQuotas(c.Request.Context(), tenant, c.GetHeader(srv.HeaderUser),
		func(quotas []srv.Quota) ([]srv.Quota, error) {
			result = q
			return replacePoolQuota(quotas, q), nil
		})
	if err != nil {
		h.writeQuotaError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// PatchQuota handles PATCH .../quotas/{pool} — replace the pool quota input.
func (h *Handler) PatchQuota(c *gin.Context) {
	tenant, pool := c.Param("tenant"), c.Param("pool")
	var req srv.PatchQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		srv.AbortWithProblem(c, http.StatusBadRequest, "InvalidBody", "request body malformed", err.Error())
		return
	}
	var result srv.Quota
	err := h.mutateQuotas(c.Request.Context(), tenant, c.GetHeader(srv.HeaderUser),
		func(quotas []srv.Quota) ([]srv.Quota, error) {
			idx := indexOfPool(quotas, pool)
			if idx == -1 {
				return nil, errQuotaNotFound
			}
			// An empty body (neither units nor quota) is a no-op that preserves
			// the existing quota rather than re-folding with no mode set — which
			// would otherwise fail ModeRequired. An explicit `units: []` still
			// zeroes the quota via the units path.
			if req.Units == nil && req.Quota == nil {
				result = quotas[idx]
				return quotas, nil
			}
			result = srv.Quota{Pool: pool, Units: req.Units, Quota: req.Quota}
			quotas[idx] = result
			return quotas, nil
		})
	if err != nil {
		h.writeQuotaError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// DeleteQuota handles DELETE .../quotas/{pool}; idempotent.
func (h *Handler) DeleteQuota(c *gin.Context) {
	tenant, pool := c.Param("tenant"), c.Param("pool")
	err := h.mutateQuotas(c.Request.Context(), tenant, c.GetHeader(srv.HeaderUser),
		func(quotas []srv.Quota) ([]srv.Quota, error) {
			out := quotas[:0]
			for _, q := range quotas {
				if q.Pool == pool {
					continue
				}
				out = append(out, q)
			}
			return out, nil
		})
	if err != nil {
		h.writeQuotaError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────── helpers

func (h *Handler) getTenant(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	return h.tenants.Get(ctx, name)
}

// foldQuotas fetches the referenced ResourcePools and folds the business-form
// selection into ElasticQuota min/max plus the round-trip annotation. An empty
// selection clears both.
func (h *Handler) foldQuotas(ctx context.Context, quotas []srv.Quota) ([]tenantv1alpha1.QuotaSpec, string, error) {
	if len(quotas) == 0 {
		return nil, "", nil
	}
	pools, err := h.fetchPools(ctx, srv.PoolNames(quotas))
	if err != nil {
		return nil, "", err
	}
	folded, err := srv.FoldQuotas(quotas, pools)
	if err != nil {
		return nil, "", err
	}
	anno, err := srv.QuotasToAnnotation(quotas)
	if err != nil {
		return nil, "", err
	}
	return folded, anno, nil
}

func (h *Handler) fetchPools(ctx context.Context, names []string) (map[string]*cmv1alpha1.ResourcePool, error) {
	out := make(map[string]*cmv1alpha1.ResourcePool, len(names))
	for _, n := range names {
		pool, err := h.pools.Get(ctx, n)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, &srv.QuotaError{Reason: srv.QuotaPoolNotFound, Pool: n}
			}
			return nil, err
		}
		out[n] = pool
	}
	return out, nil
}

// mutateWithRetry runs `mutate` against a fresh Tenant read; on a 409
// resourceVersion conflict it re-reads and reruns the closure with backoff.
// A freshly-created tenant is written repeatedly by the operator's initial
// reconcile, so a single retry loses the race — RetryOnConflict backs off
// across several attempts before surfacing the conflict.
func (h *Handler) mutateWithRetry(ctx context.Context, name string, mutate func(*tenantv1alpha1.Tenant) error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := h.getTenant(ctx, name)
		if err != nil {
			return err
		}
		base := cr.DeepCopy()
		if err := mutate(cr); err != nil {
			return err
		}
		return h.tenants.Patch(ctx, cr, base)
	})
}

// mutateQuotas loads the tenant, runs `transform` on its business-form quotas,
// re-folds the result into spec.quotas[] + the round-trip annotation, and
// patches with optimistic-lock retry + backoff (a freshly-created tenant is
// still being reconciled, so one retry loses the race).
func (h *Handler) mutateQuotas(ctx context.Context, name, user string, transform func([]srv.Quota) ([]srv.Quota, error)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := h.getTenant(ctx, name)
		if err != nil {
			return err
		}
		base := cr.DeepCopy()
		next, err := transform(srv.TenantToAPI(cr).Quotas)
		if err != nil {
			return err
		}
		folded, anno, err := h.foldQuotas(ctx, next)
		if err != nil {
			return err
		}
		cr.Spec.Quotas = folded
		setQuotaAnnotation(cr, anno)
		if user != "" {
			if cr.Annotations == nil {
				cr.Annotations = map[string]string{}
			}
			cr.Annotations[srv.LastModifiedByAnnotation] = user
		}
		return h.tenants.Patch(ctx, cr, base)
	})
}

func setQuotaAnnotation(cr *tenantv1alpha1.Tenant, anno string) {
	if anno == "" {
		if cr.Annotations != nil {
			delete(cr.Annotations, srv.QuotasAnnotation)
		}
		return
	}
	if cr.Annotations == nil {
		cr.Annotations = map[string]string{}
	}
	cr.Annotations[srv.QuotasAnnotation] = anno
}

func indexOfPool(quotas []srv.Quota, pool string) int {
	for i := range quotas {
		if quotas[i].Pool == pool {
			return i
		}
	}
	return -1
}

func replacePoolQuota(quotas []srv.Quota, q srv.Quota) []srv.Quota {
	for i := range quotas {
		if quotas[i].Pool == q.Pool {
			quotas[i] = q
			return quotas
		}
	}
	return append(quotas, q)
}

var errQuotaNotFound = errors.New("quota for pool not found")

func (h *Handler) writeQuotaError(c *gin.Context, err error) {
	var qe *srv.QuotaError
	switch {
	case errors.Is(err, errQuotaNotFound):
		srv.AbortWithProblem(c, http.StatusNotFound, "QuotaNotFound", err.Error(), "")
	case errors.As(err, &qe):
		status := http.StatusUnprocessableEntity
		if isBadQuotaInput(qe.Reason) {
			status = http.StatusBadRequest
		}
		srv.AbortWithProblem(c, status, "InvalidQuota", qe.Error(), "")
	default:
		writeK8sError(c, err, c.Param("tenant"))
	}
}

func isBadQuotaInput(reason srv.QuotaErrorReason) bool {
	switch reason {
	case srv.QuotaBadQuantity,
		srv.QuotaDuplicatePool,
		srv.QuotaModeConflict,
		srv.QuotaModeRequired,
		srv.QuotaMaxRequired,
		srv.QuotaNegativeResource,
		srv.QuotaMinWithoutMax,
		srv.QuotaMinExceedsMax:
		return true
	default:
		return false
	}
}

func tenantGroupResource() schema.GroupResource {
	return tenantv1alpha1.GroupVersion.WithResource("tenants").GroupResource()
}

func writeK8sError(c *gin.Context, err error, name string) {
	switch {
	case errors.Is(err, extensions.ErrCapabilityUnavailable):
		srv.AbortWithProblem(c, http.StatusConflict, "CapabilityUnavailable",
			"operation not supported in this deployment form", name)
	case apierrors.IsAlreadyExists(err):
		srv.AbortWithProblem(c, http.StatusConflict, "AlreadyExists", "resource already exists", name)
	case apierrors.IsNotFound(err):
		srv.AbortWithProblem(c, http.StatusNotFound, "NotFound", "resource not found", name)
	case apierrors.IsConflict(err):
		srv.AbortWithProblem(c, http.StatusConflict, "OptimisticLockConflict", "resource was modified concurrently — retry", name)
	case apierrors.IsInvalid(err):
		srv.AbortWithProblem(c, http.StatusUnprocessableEntity, "Invalid", "K8s API rejected the resource", err.Error())
	case apierrors.IsBadRequest(err):
		srv.AbortWithProblem(c, http.StatusBadRequest, "BadRequest", err.Error(), "")
	default:
		srv.AbortWithProblem(c, http.StatusInternalServerError, "K8sError", "unexpected error from Kubernetes API", err.Error())
	}
}
