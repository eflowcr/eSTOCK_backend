package tools

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlserver"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs all pending migrations. migrationURL is the source (e.g. "file://db/migrations");
// dbURL is the database connection URL (postgres://... or sqlserver://...).
// Returns nil on success or when there are no pending migrations (ErrNoChange).
func RunMigrations(migrationURL, dbURL string) error {
	m, err := migrate.New(migrationURL, dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
