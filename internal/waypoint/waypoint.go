// Package waypoint wires the application's dependencies together.
package waypoint

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/waypoint-group/waypoint-be/db/migrations"
	"github.com/waypoint-group/waypoint-be/internal/db"
)

// Waypoint application encapsulates the following:
//   - database
//   - services
//   - HTTP handler
type Waypoint struct {
	// Database is the application's PostgreSQL database connection.
	Database *db.Database
	Services *Services
	// Handler serves the application's HTTP API.
	Handler http.Handler
}

// New constructs a Waypoint application from the supplied configuration.
func New(cfg *Config) (*Waypoint, error) {
	if cfg.Migrate {
		if err := migrations.Up(cfg.DatabaseURL); err != nil {
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	services := newServices(database, cfg)
	router := newRouter(database, services, cfg)
	return &Waypoint{
		Database: database,
		Services: services,
		Handler:  router,
	}, nil
}

func (w *Waypoint) Close() {
	w.Services.Close()
	w.Database.Close()
}
