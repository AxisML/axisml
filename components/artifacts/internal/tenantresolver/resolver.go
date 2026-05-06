package tenantresolver

import (
	"context"
	"errors"

	"gorm.io/gorm"

	apperrors "github.com/axisml/axisml/components/artifacts/pkg/errors"
)

// Status values mirrored from compute's tenant lifecycle. Only Active is
// considered usable.
const (
	StatusActive    = "Active"
	StatusSuspended = "Suspended"
)

// Resolver loads tenant rows from the shared PG database. Read-only; never
// writes to the table. The package is the only place artifacts touches
// compute's schema, so a future cross-org split swaps the implementation
// here without touching call sites.
type Resolver struct {
	db *gorm.DB
}

// New returns a Resolver bound to the supplied GORM DB.
func New(db *gorm.DB) *Resolver { return &Resolver{db: db} }

// Resolve looks up a tenant by name. Returns:
//   - apperrors.CodeNotFound if absent or soft-deleted.
//   - apperrors.CodePrecondition if present but non-Active.
func (r *Resolver) Resolve(ctx context.Context, name string) (Tenant, error) {
	if name == "" {
		return Tenant{}, apperrors.New(apperrors.CodeValidation, "tenant name is required")
	}
	var t Tenant
	err := r.db.WithContext(ctx).
		Where("name = ? AND deleted_at IS NULL", name).
		Take(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Tenant{}, apperrors.Newf(apperrors.CodeNotFound, "tenant %q not found", name)
	}
	if err != nil {
		return Tenant{}, apperrors.Wrap(apperrors.CodeInternal, "lookup tenant", err)
	}
	if t.Status != StatusActive {
		return Tenant{}, apperrors.Newf(
			apperrors.CodePrecondition,
			"tenant %q is %s, not Active", name, t.Status,
		)
	}
	return t, nil
}
