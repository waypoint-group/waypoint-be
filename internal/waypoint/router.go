package waypoint

import (
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/health"
	"github.com/waypoint-group/waypoint-be/internal/middleware"
	"github.com/waypoint-group/waypoint-be/internal/user"
)

// newRouter returns the HTTP handler containing all Waypoint API routes.
func newRouter(database *db.Database, jwtConfig middleware.JWTConfig) http.Handler {
	mux := http.NewServeMux()
	const maxRequestBodyBytes = 1 << 20

	// Health
	healthH := health.NewHandler(database)
	mux.HandleFunc("GET /healthz", healthH.Healthz)
	mux.HandleFunc("GET /readyz", healthH.Readyz)

	// Users
	userH := user.NewHandler(user.NewService(database), jwtConfig)
	mux.Handle("POST /users", http.MaxBytesHandler(http.HandlerFunc(userH.CreateUser), maxRequestBodyBytes))
	// TODO: ListUsers should be admin only.
	mux.HandleFunc("GET /users", userH.ListUsers)
	mux.HandleFunc("GET /users/{id}", userH.GetUser)
	mux.HandleFunc("GET /me", userH.Me)

	return mux
}
