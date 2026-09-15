// Package services provides a collection of business logic implementations.
package services

import "github.com/waypoint-group/waypoint-be/internal/db"

// Services groups the application's domain services.
type Services struct {
	Accounts  *AccountsService
	Messaging *MessagingService
}

// New constructs the application's domain services.
func New(database *db.Database) *Services {
	return &Services{
		Accounts:  NewAccountsService(database),
		Messaging: NewMessagingService(database),
	}
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

// InvalidInputError indicates that an operation input failed validation.
type InvalidInputError struct {
	// What describes the invalid input.
	What string
}

// Error returns the input validation message.
func (e InvalidInputError) Error() string { return e.What }
