package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// AccountsService manages user registration and profile retrieval.
type AccountsService struct {
	database *db.Database
}

func NewAccountsService(database *db.Database) *AccountsService {
	return &AccountsService{
		database: database,
	}
}

// RegisterUserInput contains input for user registration.
type RegisterUserInput struct {
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

// RegisteredUser represents a successfully registered user.
type RegisteredUser struct {
	// ID is the unique identifier of the registered user.
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

// RegisterUser atomically creates a profile and its external identity mapping.
func (s *AccountsService) RegisterUser(ctx context.Context, input RegisterUserInput) (_ *RegisteredUser, registerUserError error) {
	email := strings.TrimSpace(input.Email)
	userName := strings.TrimSpace(input.UserName)
	displayName := strings.TrimSpace(input.DisplayName)
	if email == "" || userName == "" || displayName == "" {
		return nil, InvalidInputError{What: "email, user name, and display name are required"}
	}

	identityIssuer := strings.TrimSpace(input.Identity.Issuer)
	identitySubject := strings.TrimSpace(input.Identity.Subject)
	if identityIssuer == "" || identitySubject == "" {
		return nil, InvalidInputError{What: "identity issuer and subject are required"}
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			registerUserError = errors.Join(
				fmt.Errorf("rollback transaction: %w", err),
				registerUserError,
			)
		}
	}()
	// Create the user...
	u, err := tx.CreateUser(ctx, sqlc.CreateUserParams{
		ID: uuid.NewV7(), Email: email, UserName: userName, DisplayName: displayName,
	})
	if err != nil {
		if db.IsUniqueViolation(err, "users_email_unique") {
			return nil, AlreadyExistsError{What: "email"}
		}
		if db.IsUniqueViolation(err, "users_user_name_unique") {
			return nil, AlreadyExistsError{What: "user_name"}
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	// ...and the user identity, in the same transaction. This
	// ensures atomicity between users and their identities.
	_, err = tx.CreateUserIdentity(ctx, sqlc.CreateUserIdentityParams{
		ID: uuid.NewV7(), UserID: u.ID,
		Issuer: input.Identity.Issuer, Subject: input.Identity.Subject,
	})
	if err != nil {
		if db.IsUniqueViolation(err, "user_identities_identity_unique") {
			return nil, AlreadyExistsError{What: "user identity"}
		}
		return nil, fmt.Errorf("create user identity: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &RegisteredUser{
		ID: u.ID, Email: u.Email, UserName: u.UserName,
		DisplayName: u.DisplayName, CreatedAt: u.CreatedAt.Time,
	}, nil
}

// ReadUser retrieves a registered user by ID.
func (s *AccountsService) ReadUser(ctx context.Context, id uuid.UUID) (*RegisteredUser, error) {
	user, err := s.database.SelectUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "user"}
		}
		return nil, fmt.Errorf("read user: %w", err)
	}
	result := registeredUser(user)
	return &result, nil
}

// ReadUserByIdentity retrieves a registered user matching both issuer and subject.
func (s *AccountsService) ReadUserByIdentity(ctx context.Context, issuer, subject string) (*RegisteredUser, error) {
	user, err := s.database.SelectUserByIdentity(ctx, sqlc.SelectUserByIdentityParams{Issuer: issuer, Subject: subject})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{What: "user"}
		}
		return nil, fmt.Errorf("read user by identity: %w", err)
	}
	result := registeredUser(user)
	return &result, nil
}

// ListUsers returns registered users ordered by ID.
func (s *AccountsService) ListUsers(ctx context.Context) ([]RegisteredUser, error) {
	users, err := s.database.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	result := make([]RegisteredUser, len(users))
	for i, user := range users {
		result[i] = registeredUser(user)
	}
	return result, nil
}

func registeredUser(user sqlc.User) RegisteredUser {
	return RegisteredUser{
		ID: user.ID, Email: user.Email, UserName: user.UserName,
		DisplayName: user.DisplayName, CreatedAt: user.CreatedAt.Time,
	}
}
