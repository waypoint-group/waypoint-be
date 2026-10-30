package user

import (
	"time"
	"uuid"
)

// UserProfile contains a registered user and their workspace memberships.
type UserProfile struct {
	User                 User
	WorkspaceMemberships []WorkspaceMembership
}

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

// WorkspaceMembership identifies a workspace the user belongs to.
type WorkspaceMembership struct {
	WorkspaceID   uuid.UUID
	WorkspaceName string
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

// DeleteUserInput identifies the account to delete and the authenticated caller.
type DeleteUserInput struct {
	// UserID identifies the local account to delete.
	UserID uuid.UUID
	// ActorID is resolved from the verified identity and must equal UserID.
	ActorID uuid.UUID
}
