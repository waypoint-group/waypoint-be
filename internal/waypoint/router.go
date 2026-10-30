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
	security := &apiSecurity{users: services.Users, jwtConfig: cfg.JWT}
	server, err := api.NewServer(handlers, security, api.WithErrorHandler(api.HandleError))
	if err != nil {
		return nil, err
	}

	// Generated router does not have implicit HEAD support. Add an adapter that
	// redirects HEAD requests as GET to the router, restoring HEAD support.
	headSupport := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			r = r.Clone(r.Context())
			r.Method = http.MethodGet
		}
		server.ServeHTTP(w, r)
	})
	maxBytes := http.MaxBytesHandler(
		headSupport,
		1<<20,
	)

	mux := http.NewServeMux()

	// Health
	mux.Handle("GET /healthz", maxBytes)
	mux.Handle("GET /readyz", maxBytes)

	// Users
	// TODO: ListUsers should be admin only.
	mux.Handle("GET /users", maxBytes)
	mux.Handle("GET /users/{id}", maxBytes)
	mux.Handle("POST /users", maxBytes)
	mux.Handle("DELETE /users/{id}", maxBytes)
	mux.Handle("GET /me", maxBytes)

	// Workspaces
	mux.Handle("POST /workspaces", maxBytes)
	mux.Handle("GET /workspaces/{workspace_id}", maxBytes)
	mux.Handle("DELETE /workspaces/{workspace_id}", maxBytes)
	mux.Handle("POST /workspaces/{workspace_id}/users/{user_id}", maxBytes)
	mux.Handle("DELETE /workspaces/{workspace_id}/users/{user_id}", maxBytes)

	return mux, nil
}
