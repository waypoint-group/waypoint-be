// Package waypoint wires the application's dependencies together.
package waypoint

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

type Waypoint struct {
	Database *db.Database
	Services *service.Services
	Handler  http.Handler
}

func New(cfg *Config) (*Waypoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	services := service.New(database)
	httpHandler := httpapi.New(database, services)

	return &Waypoint{
		Database: database,
		Services: services,
		Handler:  httpHandler.Routes(),
	}, nil
}
