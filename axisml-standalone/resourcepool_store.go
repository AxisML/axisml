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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/validation/field"

	cmv1alpha1 "github.com/axisml/axisml/axisml-system/apis/resourcepool/v1alpha1"
	cmext "github.com/axisml/axisml/axisml-system/cluster-manager/pkg/extensions"
	apperrors "github.com/axisml/axisml/axisml-system/compute-service/pkg/errors"
	csext "github.com/axisml/axisml/axisml-system/compute-service/pkg/extensions"
)

// resourcePoolRecord persists the ResourcePool CR-shaped desired state owned by
// standalone. Tombstones keep deleted bootstrap pools from reappearing after a
// restart while preserving Kubernetes-style explicit name reuse.
type resourcePoolRecord struct {
	Name            string         `gorm:"column:name;primaryKey;size:253"`
	Object          datatypes.JSON `gorm:"column:object;type:jsonb;not null"`
	ResourceVersion int64          `gorm:"column:resource_version;not null"`
	Deleted         bool           `gorm:"column:deleted;not null;default:false"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null"`
}

func (resourcePoolRecord) TableName() string { return "standalone_resource_pools" }

// persistentResourcePoolStore is both the writable Cluster Manager provider
// and Compute's live ResourceResolver. PostgreSQL is therefore the single
// source of truth for pool CRUD and subsequent workload resource expansion.
type persistentResourcePoolStore struct {
	db *gorm.DB
}

var _ cmext.ResourcePoolProvider = (*persistentResourcePoolStore)(nil)
var _ csext.ResourceResolver = (*persistentResourcePoolStore)(nil)

func newPersistentResourcePoolStore(db *gorm.DB) *persistentResourcePoolStore {
	return &persistentResourcePoolStore{db: db}
}

// Seed imports CR YAML only when a name has never existed. In particular, it
// does not revive pools deleted through the API.
func (s *persistentResourcePoolStore) Seed(ctx context.Context, pools ...*cmv1alpha1.ResourcePool) error {
	for _, pool := range pools {
		_, record, err := newResourcePoolRecord(pool, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("seed resource pool: %w", err)
		}
		result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(record)
		if result.Error != nil {
			return fmt.Errorf("seed resource pool %q: %w", pool.Name, result.Error)
		}
	}
	return nil
}

func (s *persistentResourcePoolStore) Get(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	record, err := s.getRecord(ctx, name)
	if err != nil {
		return nil, err
	}
	return resourcePoolFromRecord(record)
}

func (s *persistentResourcePoolStore) List(ctx context.Context, opts metav1.ListOptions) (*cmv1alpha1.ResourcePoolList, error) {
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

	var records []resourcePoolRecord
	if err := s.db.WithContext(ctx).
		Where("deleted = ?", false).
		Order("name ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]cmv1alpha1.ResourcePool, 0, len(records))
	for i := range records {
		if records[i].Name <= continueAfter {
			continue
		}
		pool, err := resourcePoolFromRecord(&records[i])
		if err != nil {
			return nil, err
		}
		if selector.Matches(labels.Set(pool.Labels)) {
			items = append(items, *pool)
		}
	}

	list := &cmv1alpha1.ResourcePoolList{}
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

func (s *persistentResourcePoolStore) Create(ctx context.Context, pool *cmv1alpha1.ResourcePool) error {
	prepared, record, err := newResourcePoolRecord(pool, time.Now().UTC())
	if err != nil {
		return err
	}
	if _, err := s.Get(ctx, prepared.Name); err == nil {
		return apierrors.NewAlreadyExists(poolGR(), prepared.Name)
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(record)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		result = s.db.WithContext(ctx).Model(&resourcePoolRecord{}).
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
			return apierrors.NewAlreadyExists(poolGR(), prepared.Name)
		}
	}
	*pool = *prepared
	return nil
}

func (s *persistentResourcePoolStore) Patch(ctx context.Context, obj, base *cmv1alpha1.ResourcePool) error {
	if obj == nil || base == nil || obj.Name == "" || obj.Name != base.Name {
		return apierrors.NewBadRequest("resource pool patch requires matching non-empty names")
	}
	expected, err := strconv.ParseInt(base.ResourceVersion, 10, 64)
	if err != nil || expected < 1 {
		return apierrors.NewConflict(poolGR(), obj.Name, errors.New("invalid base resourceVersion"))
	}

	next := obj.DeepCopy()
	if err := validatePersistentResourcePool(next); err != nil {
		return err
	}
	next.ResourceVersion = strconv.FormatInt(expected+1, 10)
	if apiequality.Semantic.DeepEqual(base.Spec, next.Spec) {
		next.Generation = base.Generation
	} else {
		next.Generation = base.Generation + 1
	}
	if next.CreationTimestamp.IsZero() {
		next.CreationTimestamp = base.CreationTimestamp
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return fmt.Errorf("encode resource pool %q: %w", next.Name, err)
	}
	result := s.db.WithContext(ctx).Model(&resourcePoolRecord{}).
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
		return apierrors.NewConflict(poolGR(), next.Name, errors.New("resourceVersion changed"))
	}
	*obj = *next
	return nil
}

func (s *persistentResourcePoolStore) Delete(ctx context.Context, name string) error {
	result := s.db.WithContext(ctx).Model(&resourcePoolRecord{}).
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
		return apierrors.NewNotFound(poolGR(), name)
	}
	return nil
}

func (s *persistentResourcePoolStore) ResolveResourcePool(ctx context.Context, name string) (*cmv1alpha1.ResourcePool, error) {
	if name == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "poolName is required")
	}
	pool, err := s.Get(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, apperrors.Newf(apperrors.CodeValidation, "resource pool %q not found", name)
	}
	return pool, err
}

func (s *persistentResourcePoolStore) ResolveResourceUnit(ctx context.Context, poolName, unitName string) (*cmv1alpha1.ResourceUnit, error) {
	if unitName == "" {
		return nil, apperrors.New(apperrors.CodeValidation, "unitName is required")
	}
	pool, err := s.ResolveResourcePool(ctx, poolName)
	if err != nil {
		return nil, err
	}
	for i := range pool.Spec.Units {
		if pool.Spec.Units[i].Name == unitName {
			unit := &cmv1alpha1.ResourceUnit{}
			pool.Spec.Units[i].DeepCopyInto(unit)
			return unit, nil
		}
	}
	return nil, apperrors.Newf(apperrors.CodeValidation, "resource unit %q not found in pool %q", unitName, poolName)
}

func (s *persistentResourcePoolStore) getRecord(ctx context.Context, name string) (*resourcePoolRecord, error) {
	var record resourcePoolRecord
	err := s.db.WithContext(ctx).
		Where("name = ? AND deleted = ?", name, false).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierrors.NewNotFound(poolGR(), name)
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func newResourcePoolRecord(pool *cmv1alpha1.ResourcePool, now time.Time) (*cmv1alpha1.ResourcePool, *resourcePoolRecord, error) {
	if pool == nil || pool.Name == "" {
		return nil, nil, apierrors.NewBadRequest("resource pool name is required")
	}
	prepared := pool.DeepCopy()
	if err := validatePersistentResourcePool(prepared); err != nil {
		return nil, nil, err
	}
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
		return nil, nil, fmt.Errorf("encode resource pool %q: %w", prepared.Name, err)
	}
	record := &resourcePoolRecord{
		Name:            prepared.Name,
		Object:          datatypes.JSON(raw),
		ResourceVersion: 1,
		CreatedAt:       prepared.CreationTimestamp.Time,
		UpdatedAt:       now,
	}
	return prepared, record, nil
}

func validatePersistentResourcePool(pool *cmv1alpha1.ResourcePool) error {
	if _, err := validatePools([]*cmv1alpha1.ResourcePool{pool}); err != nil {
		return apierrors.NewInvalid(
			cmv1alpha1.GroupVersion.WithKind("ResourcePool").GroupKind(),
			pool.Name,
			field.ErrorList{field.Invalid(field.NewPath("spec"), pool.Spec, err.Error())},
		)
	}
	return nil
}

func resourcePoolFromRecord(record *resourcePoolRecord) (*cmv1alpha1.ResourcePool, error) {
	var pool cmv1alpha1.ResourcePool
	if err := json.Unmarshal(record.Object, &pool); err != nil {
		return nil, fmt.Errorf("decode resource pool %q: %w", record.Name, err)
	}
	pool.ResourceVersion = strconv.FormatInt(record.ResourceVersion, 10)
	return &pool, nil
}
