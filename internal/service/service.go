// Package service provides application business logic.
package service

import (
	"context"

	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Services groups the application's domain services.
type Services struct {
	Users           *UserService
	UserIdentities  *UserIdentityService
	Channels        *ChannelService
	ChannelMessages *ChannelMessageService
	DirectMessages  *DirectMessageService
}

// New constructs the application's domain services.
func New(database *db.Database) *Services {
	return &Services{
		Users:           NewUserService(database),
		UserIdentities:  NewUserIdentityService(database),
		Channels:        NewChannelService(database),
		ChannelMessages: NewChannelMessageService(database),
		DirectMessages:  NewDirectMessageService(database),
	}
}

func InTx(
	ctx context.Context,
	database *db.Database,
	fn func(s *Services) (any, error),
) (any, error) {
	return database.InTx(ctx, func(q *sqlc.Queries) (any, error) {
		s := &Services{
			Users:           NewUserService(q),
			UserIdentities:  NewUserIdentityService(q),
			Channels:        NewChannelService(q),
			ChannelMessages: NewChannelMessageService(q),
			DirectMessages:  NewDirectMessageService(q),
		}
		return fn(s)
	})
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
