//go:build integration

// Package testlib provides integration testing utilities for the waypoint backend.
package testlib

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// TestEnvironment owns shared containers. Close it after every instance has closed.
type TestEnvironment struct {
	postgres *Postgres
	keycloak *Keycloak
}

func NewEnvironment() (*TestEnvironment, error) {
	e := &TestEnvironment{}
	var err error

	e.postgres, err = NewPostgres()
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	e.keycloak, err = NewKeycloak()
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("failed to start keycloak: %w", err),
			e.Close(),
		)
	}

	return e, nil
}

func (e *TestEnvironment) Keycloak() *Keycloak {
	return e.keycloak
}

func (e *TestEnvironment) CreateWaypointInstance(cfg *waypoint.Config) (*TestWaypoint, error) {
	if cfg == nil {
		return nil, errors.New("waypoint configuration is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := e.postgres.CreateDatabase(ctx)
	if err != nil {
		return nil, err
	}

	w := &TestWaypoint{database: database}
	instanceCfg := *cfg
	instanceCfg.DatabaseURL = database.URL
	w.waypoint, err = waypoint.New(&instanceCfg)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to run Waypoint: %w", err), w.Close())
	}
	w.apiServer = httptest.NewServer(w.waypoint.Handler)
	return w, nil
}

func (e *TestEnvironment) Close() error {
	// Each container termination has its own 30-second deadline.
	var closeErr error
	if e.keycloak != nil {
		closeErr = errors.Join(closeErr, e.keycloak.Close())
	}
	if e.postgres != nil {
		closeErr = errors.Join(closeErr, e.postgres.Close())
	}
	return closeErr
}

type TestWaypoint struct {
	database  *TestDatabase
	waypoint  *waypoint.Waypoint
	apiServer *httptest.Server
}

func (w *TestWaypoint) Client() *http.Client {
	return w.apiServer.Client()
}

func (w *TestWaypoint) Request(
	ctx context.Context, method string, route string, body io.Reader,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, w.apiServer.URL+route, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return w.Client().Do(request)
}

func (w *TestWaypoint) RequestWithAccessToken(
	ctx context.Context, method string, route string, body io.Reader, accessToken string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, w.apiServer.URL+route, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	return w.Client().Do(request)
}

func (w *TestWaypoint) Close() error {
	if w.apiServer != nil {
		w.apiServer.Close()
	}
	if w.waypoint != nil {
		w.waypoint.Close()
	}
	if w.database != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return w.database.Close(ctx)
	}
	return nil
}
