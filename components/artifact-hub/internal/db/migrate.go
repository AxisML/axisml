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

// Migrate applies all pending migrations using golang-migrate. PG advisory
// locks make it safe to run concurrently from multiple replicas.
func Migrate(db *gorm.DB) error {
	src, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	// artifact-hub shares the infra Postgres database with compute-service.
	// golang-migrate's default version table is "schema_migrations"; if both
	// services used it they would collide (whoever migrates first sets the
	// version, and the other sees ErrNoChange and skips its own tables). Use a
	// service-scoped version table so each tracks its migrations independently.
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{
		MigrationsTable: "artifact_hub_schema_migrations",
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
