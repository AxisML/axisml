// Package db opens the Platform PostgreSQL connection and applies migrations.
package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/axisml/axisml/components/platform/internal/config"
)

// Open returns a GORM DB connected to PostgreSQL.
func Open(cfg config.Config) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		// Translate driver errors (e.g. unique_violation → gorm.ErrDuplicatedKey)
		// so services can map racing duplicate inserts to 409 instead of 500.
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return gormDB, nil
}

// Ping issues a context-aware health check.
func Ping(ctx context.Context, gormDB *gorm.DB) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
