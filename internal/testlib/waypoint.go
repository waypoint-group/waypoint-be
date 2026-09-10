//go:build integration

// Package testlib provides integration testing utilities for the waypoint backend.
package testlib

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

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
}

func (w *Waypoint) URL() string {
	return w.httpServer.URL
}

func (w *Waypoint) Get(route string) (*http.Response, error) {
	response, err := http.Get(w.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}
	return response, nil
}

func (w *Waypoint) Post(route string, body io.Reader) (*http.Response, error) {
	response, err := http.Post(w.URL(), "application/json", body)
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}
	return response, nil
}
