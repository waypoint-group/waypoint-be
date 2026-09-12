// Package httpapi provides the HTTP transport for the Waypoint service.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

const maxRequestBodyBytes = 1 << 20

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

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return fmt.Errorf("decode trailing request data: %w", err)
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(append(payload, '\n'))
	return err
}

func writeInternalServerError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func writeInvalidRequestBody(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
}
