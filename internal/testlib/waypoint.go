//go:build integration

// Package testlib provides integration testing utilities for the waypoint backend.
package testlib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

var waypointPort = 8080

type Waypoint struct {
	postgres   *Postgres
	waypoint   *waypoint.Waypoint
	httpServer *httptest.Server
}

func NewWaypoint() (*Waypoint, error) {
	postgres, err := NewPostgres()
	if err != nil {
		return nil, fmt.Errorf("failed to run Postgres: %w", err)
	}

	waypoint, err := waypoint.New(
		&waypoint.Config{
			Port:        waypointPort,
			DatabaseURL: postgres.URL,
			// Always run migrations for tests.
			Migrate: true,
		},
	)
	if err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = postgres.Terminate(ctx)
		return nil, fmt.Errorf("failed to run Waypoint: %w", err)
	}

	return &Waypoint{
		postgres:   postgres,
		waypoint:   waypoint,
		httpServer: httptest.NewServer(waypoint.Handler),
	}, nil
}

func (w *Waypoint) Close() {
	w.httpServer.Close()
	w.waypoint.Database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = w.postgres.Terminate(ctx)
}

func (w *Waypoint) BaseURL() string {
	return w.httpServer.URL
}

func (w *Waypoint) RouteURL(route string) string {
	return fmt.Sprintf("%v%v", w.BaseURL(), route)
}

func (w *Waypoint) Get(route string) (*http.Response, error) {
	response, err := http.Get(w.RouteURL(route))
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	return response, nil
}

func (w *Waypoint) Post(route string, body io.Reader) (*http.Response, error) {
	response, err := http.Post(w.RouteURL(route), "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}
	return response, nil
}
