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
	"sync"
	"time"

	"github.com/waypoint-group/waypoint-be/internal/service"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// TestEnvironment owns shared containers. Close it after every instance has closed.
// The zero value is ready to use; dependency startup failures are cached.
type TestEnvironment struct {
	mu              sync.Mutex
	postgres        *Postgres
	postgresStarted bool
	postgresErr     error
	keycloak        *Keycloak
	keycloakStarted bool
	keycloakErr     error
	closed          bool
	closeErr        error
}

func NewEnvironment() (*TestEnvironment, error) {
	return &TestEnvironment{}, nil
}

func (e *TestEnvironment) Postgres() (*Postgres, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, errors.New("test environment is closed")
	}
	if !e.postgresStarted {
		e.postgresStarted = true
		e.postgres, e.postgresErr = NewPostgres()
	}
	return e.postgres, e.postgresErr
}

func (e *TestEnvironment) Keycloak() (*Keycloak, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return nil, errors.New("test environment is closed")
	}
	if !e.keycloakStarted {
		e.keycloakStarted = true
		e.keycloak, e.keycloakErr = NewKeycloak()
	}
	return e.keycloak, e.keycloakErr
}

func (e *TestEnvironment) CreateWaypointInstance(cfg *waypoint.Config) (*TestWaypoint, error) {
	if cfg == nil {
		return nil, errors.New("waypoint configuration is required")
	}

	pg, err := e.Postgres()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := pg.CreateDatabase(ctx)
	if err != nil {
		return nil, err
	}

	w := &TestWaypoint{database: database}
	instanceCfg := *cfg
	instanceCfg.DatabaseURL = database.URL
	instanceCfg.Migrate = true
	w.waypoint, err = waypoint.New(&instanceCfg)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to run Waypoint: %w", err), w.Close())
	}
	w.apiServer = httptest.NewServer(w.waypoint.Handler)
	return w, nil
}

func (e *TestEnvironment) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return e.closeErr
	}
	e.closed = true
	// Each container termination has its own 30-second deadline.
	if e.keycloak != nil {
		e.closeErr = errors.Join(e.closeErr, e.keycloak.Close())
	}
	if e.postgres != nil {
		e.closeErr = errors.Join(e.closeErr, e.postgres.Close())
	}
	return e.closeErr
}

type TestWaypoint struct {
	database  *TestDatabase
	waypoint  *waypoint.Waypoint
	apiServer *httptest.Server
}

func (w *TestWaypoint) Services() *service.Services {
	return w.waypoint.Services
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
