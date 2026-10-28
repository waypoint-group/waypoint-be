package health

import (
	"context"
	"strings"

	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/db"
)

type Handler struct{ database *db.Database }

func NewHandler(database *db.Database) *Handler { return &Handler{database: database} }

func (h *Handler) Healthz(context.Context) (api.HealthzOK, error) {
	return api.HealthzOK{Data: strings.NewReader("OK")}, nil
}

func (h *Handler) Readyz(ctx context.Context) (api.ReadyzRes, error) {
	if err := h.database.Ping(ctx); err != nil {
		return &api.ServiceUnavailable{Data: strings.NewReader(ErrDatabaseUnavailable.Error() + "\n")}, nil
	}
	return &api.ReadyzOK{Data: strings.NewReader("READY")}, nil
}
