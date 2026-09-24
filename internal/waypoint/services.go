package waypoint

import (
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/user"
	"github.com/waypoint-group/waypoint-be/internal/workspace"
)

type Services struct {
	Users      *user.Service
	Workspaces *workspace.Service
}

func newServices(database *db.Database, cfg *Config) *Services {
	return &Services{
		Users:      user.NewService(database),
		Workspaces: workspace.NewService(database, cfg.WorkspaceCleanupInterval),
	}
}

func (s *Services) Close() {
	s.Workspaces.Close()
}
