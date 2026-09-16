package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Service manages user registration and profile retrieval.
type Service struct {
	database *db.Database
}

func NewService(database *db.Database) *Service {
	return &Service{
		database: database,
	}
}

// CreateUser atomically creates a profile and its external identity mapping.
func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (_ *User, createUserError error) {
	email := strings.TrimSpace(input.Email)
	userName := strings.TrimSpace(input.UserName)
	displayName := strings.TrimSpace(input.DisplayName)
	if email == "" {
		return nil, ErrInvalidEmail
	}
	if userName == "" {
		return nil, ErrInvalidUserName
	}
	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}

	identityIssuer := strings.TrimSpace(input.Identity.Issuer)
	identitySubject := strings.TrimSpace(input.Identity.Subject)
	if identityIssuer == "" || identitySubject == "" {
		return nil, ErrInvalidIdentity
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			createUserError = errors.Join(
				fmt.Errorf("rollback transaction: %w", err),
				createUserError,
			)
		}
	}()
	// Create the user...
	u, err := tx.CreateUser(ctx, sqlc.CreateUserParams{
		ID: uuid.NewV7(), Email: email, UserName: userName, DisplayName: displayName,
	})
	if err != nil {
		if db.IsUniqueViolation(err, "users_email_unique") {
			return nil, ErrEmailTaken
		}
		if db.IsUniqueViolation(err, "users_user_name_unique") {
			return nil, ErrUserNameTaken
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
			return nil, ErrIdentityTaken
		}
		return nil, fmt.Errorf("create user identity: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &User{
		ID: u.ID, Email: u.Email, UserName: u.UserName,
		DisplayName: u.DisplayName, CreatedAt: u.CreatedAt.Time,
	}, nil
}

// ReadUser retrieves a registered user by ID.
func (s *Service) ReadUser(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.database.SelectUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read user: %w", err)
	}
	result := userFromRow(user)
	return &result, nil
}

// ReadUserByIdentity retrieves a registered user matching both issuer and subject.
func (s *Service) ReadUserByIdentity(ctx context.Context, issuer, subject string) (*User, error) {
	user, err := s.database.SelectUserByIdentity(ctx, sqlc.SelectUserByIdentityParams{Issuer: issuer, Subject: subject})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read user by identity: %w", err)
	}
	result := userFromRow(user)
	return &result, nil
}

// ListUsers returns registered users ordered by ID.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	users, err := s.database.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	result := make([]User, len(users))
	for i, user := range users {
		result[i] = userFromRow(user)
	}
	return result, nil
}

func userFromRow(row sqlc.User) User {
	return User{
		ID:          row.ID,
		Email:       row.Email,
		UserName:    row.UserName,
		DisplayName: row.DisplayName,
		CreatedAt:   row.CreatedAt.Time,
	}
}
