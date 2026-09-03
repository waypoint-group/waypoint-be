// Package service provides application business logic.
package service

import (
	"github.com/waypoint-group/waypoint-be/internal/db"
)

// Services groups the application's domain services.
type Services struct {
	Users    *UserService
	Channels *ChannelService
	Messages *MessageService
}

type NotFoundError struct {
	What string
}

func (e NotFoundError) Error() string {
	return e.What + " not found"
}

type AlreadyExistsError struct {
	What string
}

func (e AlreadyExistsError) Error() string {
	return e.What + " already exists"
}

// New constructs the application's domain services.
func New(database *db.Database) *Services {
	return &Services{
		Users:    NewUserService(database),
		Channels: NewChannelService(database),
		Messages: NewMessageService(database),
	}
}
