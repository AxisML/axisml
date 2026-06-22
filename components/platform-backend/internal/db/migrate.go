package db

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrate applies all pending migrations. PG advisory locks make it safe to run
// from multiple replicas concurrently.
func Migrate(gormDB *gorm.DB) error {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return err
	}
	// Platform shares the `axisml` database with compute-service / artifact-hub;
	// each service tracks its own migrations in a distinct table to avoid version
	// collisions on the default schema_migrations table.
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "platform_schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("postgres driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
