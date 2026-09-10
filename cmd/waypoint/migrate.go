package main

import (
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/waypoint-group/waypoint-be/db/migrations"
)

// MigrateCommand configures the database migration command.
type MigrateCommand struct {
	DatabaseURL string `name:"database-url" env:"WAYPOINT_DATABASE_URL" required:"" help:"PostgreSQL connection URL"`
}

// Run applies all pending database migrations.
func (cmd *MigrateCommand) Run() (runErr error) {
	return Migrate(cmd.DatabaseURL)
}

func Migrate(databaseURL string) (migrateErr error) {
	return migrations.Up(databaseURL)
}
