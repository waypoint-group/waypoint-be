package health

import (
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/db"
)

// Handler serves application health and readiness checks.
type Handler struct {
	database *db.Database
}

// NewHandler constructs health checks backed by the application database.
func NewHandler(database *db.Database) *Handler {
	return &Handler{database: database}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.database.Ping(r.Context()); err != nil {
		http.Error(w, ErrDatabaseUnavailable.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("READY"))
}
