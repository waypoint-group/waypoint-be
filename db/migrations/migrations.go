// Package migrations provides access to database migration files
// as well as utilities to run them automatically against a database.
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// migrationFS contains the embedded SQL migrations.
//
//go:embed *.sql
var migrationFS embed.FS

// Up looks at the embedded file system and runs any pending migrations
// against the database.
func Up(databaseURL string) (migrateErr error) {
	sourceName := "migration-fs"
	sourceInstance, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance(
		sourceName,
		sourceInstance,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}
	defer func() {
		closeErr := errors.Join(m.Close())
		if closeErr != nil {
			// Preserve existing migration errors.
			migrateErr = errors.Join(
				migrateErr,
				fmt.Errorf("close migration source: %w", closeErr),
			)
		}
	}()

	log.Println("running database migrations")

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
