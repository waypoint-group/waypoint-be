package service

import "github.com/waypoint-group/waypoint-be/internal/db"

type MessageService struct {
	database *db.Database
}

func NewMessageService(database *db.Database) *MessageService {
	return &MessageService{database: database}
}
