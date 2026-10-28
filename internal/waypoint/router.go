package waypoint

import (
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/health"
	"github.com/waypoint-group/waypoint-be/internal/user"
	"github.com/waypoint-group/waypoint-be/internal/workspace"
)

type (
	healthHandler    = health.Handler
	userHandler      = user.Handler
	workspaceHandler = workspace.Handler
)

// apiHandler combines the domain handlers into ogen's operation interface.
type apiHandler struct {
	*healthHandler
	*userHandler
	*workspaceHandler
}

func newRouter(database *db.Database, services *Services, cfg *Config) (*http.ServeMux, error) {
	handlers := &apiHandler{health.NewHandler(database), user.NewHandler(services.Users), workspace.NewHandler(services.Workspaces)}
	server, err := api.NewServer(handlers, services.Users, api.WithErrorHandler(api.HandleError))
	if err != nil {
		return nil, err
	}
	maxBytes := http.MaxBytesHandler(server, 1<<20)
	requireIdentity := user.RequireIdentity(cfg.JWT)(maxBytes)
	requireActor := services.Users.RequireActor(cfg.JWT)(maxBytes)

	mux := http.NewServeMux()

	// Health
	mux.Handle("GET /healthz", maxBytes)
	mux.Handle("GET /readyz", maxBytes)

	// Users
	// TODO: ListUsers should be admin only.
	mux.Handle("GET /users", maxBytes)
	mux.Handle("GET /users/{id}", maxBytes)
	mux.Handle("POST /users", requireIdentity)
	mux.Handle("DELETE /users/{id}", requireActor)
	mux.Handle("GET /me", requireIdentity)

	// Workspaces
	mux.Handle("POST /workspaces", requireActor)
	mux.Handle("GET /workspaces/{workspace_id}", requireActor)
	mux.Handle("DELETE /workspaces/{workspace_id}", requireActor)
	mux.Handle("POST /workspaces/{workspace_id}/users/{user_id}", requireActor)
	mux.Handle("DELETE /workspaces/{workspace_id}/users/{user_id}", requireActor)

	return mux, nil
}
