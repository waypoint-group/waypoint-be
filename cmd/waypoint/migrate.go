package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type MigrateCommand struct {
	DatabaseURL string `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL"`
}

func (cmd *MigrateCommand) Run() (runErr error) {
	return Migrate(cmd.DatabaseURL)
}

func Migrate(databaseURL string) (migrateErr error) {
	m, err := migrate.New("file://db/migrations", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}
	defer func() {
		closeErr := errors.Join(m.Close())
		if closeErr != nil {
			migrateErr = fmt.Errorf("close migration source: %w", closeErr)
		}
	}()

	log.Println("running database migrations")

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
