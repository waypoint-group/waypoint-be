package service

import "github.com/waypoint-group/waypoint-be/internal/db"

// ChannelService provides business operations for channels.
type ChannelService struct {
	database *db.Database
}

func NewChannelService(database *db.Database) *ChannelService {
	return &ChannelService{database: database}
}
