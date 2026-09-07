// Package service provides application business logic.
package service

import (
	"github.com/waypoint-group/waypoint-be/internal/db"
)

// Services groups the application's domain services.
type Services struct {
	Users           *UserService
	Channels        *ChannelService
	ChannelMessages *ChannelMessageService
	DirectMessages  *DirectMessageService
}

// NotFoundError indicates that a requested domain resource does not exist.
type NotFoundError struct {
	// What identifies the resource that was not found.
	What string
}

// Error returns a human-readable not-found message.
func (e NotFoundError) Error() string {
	return e.What + " not found"
}

// AlreadyExistsError indicates that a domain resource conflicts with an existing resource.
type AlreadyExistsError struct {
	// What identifies the resource that already exists.
	What string
}

// Error returns a human-readable conflict message.
func (e AlreadyExistsError) Error() string {
	return e.What + " already exists"
}

// New constructs the application's domain services.
func New(database *db.Database) *Services {
	return &Services{
		Users:           NewUserService(database),
		Channels:        NewChannelService(database),
		ChannelMessages: NewChannelMessageService(database),
		DirectMessages:  NewDirectMessageService(database),
	}
}
