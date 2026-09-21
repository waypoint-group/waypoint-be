package waypoint

import (
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/health"
	"github.com/waypoint-group/waypoint-be/internal/user"
	"github.com/waypoint-group/waypoint-be/internal/workspace"
)

// router owns the HTTP routes and the services that require shutdown.
type router struct {
	*http.ServeMux
	workspaceService *workspace.Service
}

func (r *router) Close() {
	r.workspaceService.Close()
}

// newRouter returns the HTTP handler containing all Waypoint API routes.
func newRouter(database *db.Database, services *Services, cfg *Config) *http.ServeMux {
	mux := http.NewServeMux()
	const maxRequestBodyBytes = 1 << 20

	// Health
	healthH := health.NewHandler(database)
	mux.HandleFunc("GET /healthz", healthH.Healthz)
	mux.HandleFunc("GET /readyz", healthH.Readyz)

	// Users
	userH := user.NewHandler(services.Users, cfg.JWT)
	mux.Handle("POST /users", http.MaxBytesHandler(http.HandlerFunc(userH.CreateUser), maxRequestBodyBytes))
	// TODO: ListUsers should be admin only.
	mux.HandleFunc("GET /users", userH.ListUsers)
	mux.HandleFunc("GET /users/{id}", userH.GetUser)
	mux.HandleFunc("GET /me", userH.Me)

	// Workspaces
	workspaceH := workspace.NewHandler(services.Workspaces, services.Users, cfg.JWT)
	mux.Handle("POST /workspaces", http.MaxBytesHandler(http.HandlerFunc(workspaceH.CreateWorkspace), maxRequestBodyBytes))
	mux.HandleFunc("GET /workspaces/{workspace_id}", workspaceH.GetWorkspace)
	mux.HandleFunc("POST /workspaces/{workspace_id}/users/{user_id}", workspaceH.AddWorkspaceMember)
	mux.HandleFunc("DELETE /workspaces/{workspace_id}/users/{user_id}", workspaceH.RemoveWorkspaceMember)
	mux.HandleFunc("DELETE /workspaces/{workspace_id}", workspaceH.DeleteWorkspace)

	return mux
}
