package user

import (
	"time"
	"uuid"
)

// User represents a successfully created user.
type User struct {
	// ID is the unique identifier of the created user.
	ID uuid.UUID
	// Email is the user's email address.
	Email string
	// UserName is the user's unique username handle.
	UserName string
	// DisplayName is the user's full or display name.
	DisplayName string
	// CreatedAt is the timestamp when the user account was created.
	CreatedAt time.Time
}

// CreateUserInput contains input for user creation.
type CreateUserInput struct {
	// Email is the user's email address.
	Email string
	// UserName is the user's unique username handle.
	UserName string
	// DisplayName is the user's full or display name.
	DisplayName string
	// Identity is the external identity provider details.
	Identity ExternalIdentity
}

// ExternalIdentity is the external identity provider details.
type ExternalIdentity struct {
	// Issuer is the authentication provider domain.
	Issuer string
	// Subject is the unique user identifier from the provider.
	Subject string
}

// CreateUserRequest contains the fields accepted when creating a user.
type CreateUserRequest struct {
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
}

// CreateUserResponse is returned after a user is created.
type CreateUserResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// GetUserResponse is returned when retrieving a user.
type GetUserResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// ListUsersResponse contains the users returned by the list endpoint.
type ListUsersResponse struct {
	// Users contains the users in the response.
	Users []GetUserResponse `json:"users"`
}

// MeResponse contains the authenticated user's profile.
type MeResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}
