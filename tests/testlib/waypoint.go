//go:build integration

// Package testlib provides integration testing utilities for the waypoint backend.
package testlib

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/waypoint-group/waypoint-be/internal/service"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

var waypointPort = 8080

type Waypoint struct {
	postgres  *Postgres
	keycloak  *Keycloak
	waypoint  *waypoint.Waypoint
	apiServer *httptest.Server
}

func NewWaypoint() (*Waypoint, error) {
	w := &Waypoint{}

	var err error
	w.postgres, err = NewPostgres()
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to run Postgres: %w", err)
	}
	w.keycloak, err = NewKeycloak()
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to run Keycloak: %w", err)
	}

	w.waypoint, err = waypoint.New(
		&waypoint.Config{
			Port:        waypointPort,
			DatabaseURL: w.postgres.URL,
			// Always run migrations for tests.
			Migrate: true,
			JWT:     w.keycloak.JWTConfig,
		},
	)
	if err != nil {
		w.Close()
		return nil, fmt.Errorf("failed to run Waypoint: %w", err)
	}

	w.apiServer = httptest.NewServer(w.waypoint.Handler)
	return w, nil
}

func (w *Waypoint) Close() {
	if w.apiServer != nil {
		w.apiServer.Close()
	}
	if w.waypoint != nil {
		w.waypoint.Close()
	}
	if w.keycloak != nil {
		w.keycloak.Close()
	}
	if w.postgres != nil {
		w.postgres.Close()
	}
}

func (w *Waypoint) Keycloak() *Keycloak {
	return w.keycloak
}

func (w *Waypoint) Services() *service.Services {
	return w.waypoint.Services
}

func (w *Waypoint) Client() *http.Client {
	return w.apiServer.Client()
}

func (w *Waypoint) Request(
	ctx context.Context, method string, route string, body io.Reader,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, w.apiServer.URL+route, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return w.Client().Do(request)
}

func (w *Waypoint) RequestWithAccessToken(
	ctx context.Context, method string, route string, body io.Reader, accessToken string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, w.apiServer.URL+route, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	return w.Client().Do(request)
}
