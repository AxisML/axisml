package standalone

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/uuid"

	tenantv1alpha1 "github.com/axisml/axisml/axisml-system/apis/tenant/v1alpha1"
	cmext "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	csext "github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// tenantRecord is the standalone persistence form of the Tenant CR. Deleted
// rows are retained as internal tombstones so a tenant supplied by bootstrap
// YAML stays deleted across restarts; a later explicit Create may reuse the
// name, matching Kubernetes hard-delete semantics at the public boundary.
type tenantRecord struct {
	Name            string         `gorm:"column:name;primaryKey;size:253"`
	Object          datatypes.JSON `gorm:"column:object;type:jsonb;not null"`
	ResourceVersion int64          `gorm:"column:resource_version;not null"`
	Deleted         bool           `gorm:"column:deleted;not null;default:false"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null"`
}

func (tenantRecord) TableName() string { return "standalone_tenants" }

// persistentTenantStore is the writable standalone Tenant provider. Tenant
// desired state lives in PostgreSQL while ResourcePools remain host-configured
// and read-only. The same store also resolves live quota changes for Compute's
// cross-runtime admission queue.
type persistentTenantStore struct {
	db          *gorm.DB
	materialize func(context.Context, *tenantv1alpha1.Tenant) error
}

var _ cmext.TenantProvider = (*persistentTenantStore)(nil)
var _ csext.QuotaResolver = (*persistentTenantStore)(nil)

// newPersistentTenantStore builds a writable Tenant store over an already-open
// database. The App keeps this backend-specific implementation behind the
// shared Cluster Manager and Compute extension interfaces.
func newPersistentTenantStore(db *gorm.DB) *persistentTenantStore {
	return &persistentTenantStore{db: db}
}

// Seed inserts startup YAML tenants only when their name has never been seen.
// It deliberately does not revive a deleted row, so an API deletion survives a
// process restart even though the original bootstrap YAML remains mounted.
func (s *persistentTenantStore) Seed(ctx context.Context, tenants ...*tenantv1alpha1.Tenant) error {
	for _, tenant := range tenants {
		_, record, err := newTenantRecord(tenant, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("seed tenant: %w", err)
		}
		result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(record)
		if result.Error != nil {
			return fmt.Errorf("seed tenant %q: %w", tenant.Name, result.Error)
		}
	}
	return nil
}

// Get returns one live tenant or a Kubernetes-shaped NotFound error so the
// shared Cluster Manager handler preserves its existing HTTP contract.
func (s *persistentTenantStore) Get(ctx context.Context, name string) (*tenantv1alpha1.Tenant, error) {
	record, err := s.getRecord(ctx, name)
	if err != nil {
		return nil, err
	}
	return tenantFromRecord(record)
}

// List applies the Cluster Manager label selector and opaque name cursor over
// a deterministic name ordering. The tenant count is expected to remain small
// on one host, so filtering decoded CR metadata in memory keeps the persisted
// shape faithful without duplicating arbitrary labels into relational columns.
func (s *persistentTenantStore) List(ctx context.Context, opts metav1.ListOptions) (*tenantv1alpha1.TenantList, error) {
	selector := labels.Everything()
	if opts.LabelSelector != "" {
		parsed, err := labels.Parse(opts.LabelSelector)
		if err != nil {
			return nil, apierrors.NewBadRequest(fmt.Sprintf("invalid label selector: %v", err))
		}
		selector = parsed
	}

	continueAfter := ""
	if opts.Continue != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(opts.Continue)
		if err != nil || len(decoded) == 0 {
			return nil, apierrors.NewBadRequest("invalid continue token")
		}
		continueAfter = string(decoded)
	}

	var records []tenantRecord
	if err := s.db.WithContext(ctx).
		Where("deleted = ?", false).
		Order("name ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]tenantv1alpha1.Tenant, 0, len(records))
	for i := range records {
		if records[i].Name <= continueAfter {
			continue
		}
		tenant, err := tenantFromRecord(&records[i])
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(tenant.Labels)) {
			items = append(items, *tenant)
		}
	}

	list := &tenantv1alpha1.TenantList{}
	limit := int(opts.Limit)
	if limit > 0 && len(items) > limit {
		list.Items = items[:limit]
		last := list.Items[len(list.Items)-1].Name
		list.Continue = base64.RawURLEncoding.EncodeToString([]byte(last))
	} else {
		list.Items = items
	}
	return list, nil
}

// Create persists a new tenant or explicitly revives a previously deleted
// name. Materialization happens first so a Docker provisioning failure never
// publishes a tenant whose declared predefined volumes are unavailable.
func (s *persistentTenantStore) Create(ctx context.Context, tenant *tenantv1alpha1.Tenant) error {
	prepared, record, err := newTenantRecord(tenant, time.Now().UTC())
	if err != nil {
		return err
	}
	if _, err := s.Get(ctx, prepared.Name); err == nil {
		return apierrors.NewAlreadyExists(tenantGR(), prepared.Name)
	} else if !apierrors.IsNotFound(err) {
		return err
	}
	if s.materialize != nil {
		if err := s.materialize(ctx, prepared); err != nil {
			return err
		}
	}

	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(record)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		result = s.db.WithContext(ctx).Model(&tenantRecord{}).
			Where("name = ? AND deleted = ?", prepared.Name, true).
			Updates(map[string]any{
				"object":           record.Object,
				"resource_version": record.ResourceVersion,
				"deleted":          false,
				"created_at":       record.CreatedAt,
				"updated_at":       record.UpdatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return apierrors.NewAlreadyExists(tenantGR(), prepared.Name)
		}
	}
	*tenant = *prepared
	return nil
}

// Patch applies an optimistic replacement using the base resourceVersion. A
// stale base returns the same Kubernetes Conflict shape as the CR-backed store,
// allowing the shared handler's RetryOnConflict loop to replay the mutation.
func (s *persistentTenantStore) Patch(ctx context.Context, obj, base *tenantv1alpha1.Tenant) error {
	if obj == nil || base == nil || obj.Name == "" || obj.Name != base.Name {
		return apierrors.NewBadRequest("tenant patch requires matching non-empty names")
	}
	expected, err := strconv.ParseInt(base.ResourceVersion, 10, 64)
	if err != nil || expected < 1 {
		return apierrors.NewConflict(tenantGR(), obj.Name, errors.New("invalid base resourceVersion"))
	}

	next := obj.DeepCopy()
	next.ResourceVersion = strconv.FormatInt(expected+1, 10)
	if apiequality.Semantic.DeepEqual(base.Spec, next.Spec) {
		next.Generation = base.Generation
	} else {
		next.Generation = base.Generation + 1
	}
	if next.CreationTimestamp.IsZero() {
		next.CreationTimestamp = base.CreationTimestamp
	}
	if s.materialize != nil {
		if err := s.materialize(ctx, next); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("encode tenant %q: %w", next.Name, err)
	}
	result := s.db.WithContext(ctx).Model(&tenantRecord{}).
		Where("name = ? AND deleted = ? AND resource_version = ?", next.Name, false, expected).
		Updates(map[string]any{
			"object":           datatypes.JSON(raw),
			"resource_version": expected + 1,
			"updated_at":       time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if _, err := s.getRecord(ctx, next.Name); err != nil {
			return err
		}
		return apierrors.NewConflict(tenantGR(), next.Name, errors.New("resourceVersion changed"))
	}
	*obj = *next
	return nil
}

// Delete hard-deletes the tenant at the public boundary while retaining an
// internal tombstone that prevents bootstrap YAML from recreating it later.
// Predefined data volumes are intentionally retained, matching Kubernetes.
func (s *persistentTenantStore) Delete(ctx context.Context, name string) error {
	result := s.db.WithContext(ctx).Model(&tenantRecord{}).
		Where("name = ? AND deleted = ?", name, false).
		Updates(map[string]any{
			"deleted":          true,
			"resource_version": gorm.Expr("resource_version + 1"),
			"updated_at":       time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apierrors.NewNotFound(tenantGR(), name)
	}
	return nil
}

// ResolveQuota returns the current hard maximum for one tenant/pool. Reads hit
// PostgreSQL on every admission pass, so quota CRUD takes effect without a
// restart or a second cache-invalidation protocol.
func (s *persistentTenantStore) ResolveQuota(ctx context.Context, tenant, pool string) (corev1.ResourceList, error) {
	if tenant == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "tenant is required")
	}
	if pool == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "pool is required")
	}
	current, err := s.Get(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("tenant %q not found: %w", tenant, err)
	}
	for _, quota := range current.Spec.Quotas {
		if quota.Pool == pool {
			return quota.Max.DeepCopy(), nil
		}
	}
	return nil, fmt.Errorf("tenant %q has no quota for resource pool %q", tenant, pool)
}

func (s *persistentTenantStore) getRecord(ctx context.Context, name string) (*tenantRecord, error) {
	var record tenantRecord
	err := s.db.WithContext(ctx).
		Where("name = ? AND deleted = ?", name, false).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierrors.NewNotFound(tenantGR(), name)
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func newTenantRecord(tenant *tenantv1alpha1.Tenant, now time.Time) (*tenantv1alpha1.Tenant, *tenantRecord, error) {
	if tenant == nil || tenant.Name == "" {
		return nil, nil, apierrors.NewBadRequest("tenant name is required")
	}
	prepared := tenant.DeepCopy()
	prepared.ResourceVersion = "1"
	if prepared.Generation == 0 {
		prepared.Generation = 1
	}
	if prepared.UID == "" {
		prepared.UID = uuid.NewUUID()
	}
	if prepared.CreationTimestamp.IsZero() {
		prepared.CreationTimestamp = metav1.NewTime(now)
	}
	raw, err := json.Marshal(prepared)
	if err != nil {
		return nil, nil, fmt.Errorf("encode tenant %q: %w", prepared.Name, err)
	}
	record := &tenantRecord{
		Name:            prepared.Name,
		Object:          datatypes.JSON(raw),
		ResourceVersion: 1,
		CreatedAt:       prepared.CreationTimestamp.Time,
		UpdatedAt:       now,
	}
	return prepared, record, nil
}

func tenantFromRecord(record *tenantRecord) (*tenantv1alpha1.Tenant, error) {
	var tenant tenantv1alpha1.Tenant
	if err := json.Unmarshal(record.Object, &tenant); err != nil {
		return nil, fmt.Errorf("decode tenant %q: %w", record.Name, err)
	}
	tenant.ResourceVersion = strconv.FormatInt(record.ResourceVersion, 10)
	return &tenant, nil
}
