package httpapi

import (
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

// Option configures an HTTP API handler.
type Option func(*Handler)

// Handler serves the Waypoint HTTP API.
type Handler struct {
	services  *service.Services
	database  *db.Database
	jwtConfig middleware.JWTConfig
}

// New constructs an HTTP API handler backed by the supplied database and services.
func New(database *db.Database, services *service.Services, options ...Option) *Handler {
	handler := &Handler{
		services: services,
		database: database,
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

// WithJWTVerification configures RS256 access token verification for authenticated routes.
// Missing issuer, audience, or key resolver causes authentication to fail closed.
func WithJWTVerification(config middleware.JWTConfig) Option {
	return func(h *Handler) { h.jwtConfig = config }
}

// Routes returns the HTTP handler containing all Waypoint API routes.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /readyz", h.Ready)

	// Users
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("GET /users", h.ListUsers)
	mux.HandleFunc("GET /users/{id}", h.GetUser)

	// User identity
	mux.HandleFunc("GET /me", h.Me)

	return mux
}
