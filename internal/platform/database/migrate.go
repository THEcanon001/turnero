package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending up migrations.
func Migrate(dsn string) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("database.migrate: open source: %w", err)
	}

	// golang-migrate expects the pgx5 scheme
	m, err := migrate.NewWithSourceInstance("iofs", source, pgxDSN(dsn))
	if err != nil {
		return fmt.Errorf("database.migrate: create instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("database.migrate: up: %w", err)
	}

	return nil
}

// pgxDSN converts a standard postgres:// DSN to pgx5:// for golang-migrate.
func pgxDSN(dsn string) string {
	if len(dsn) > 11 && dsn[:11] == "postgres://" {
		return "pgx5://" + dsn[11:]
	}
	return dsn
}
