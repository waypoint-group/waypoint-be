package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
)

// UserStore defines the data access interface for user operations.
type UserStore interface {
	CreateUser(context.Context, sqlc.CreateUserParams) (sqlc.User, error)
	SelectUser(context.Context, uuid.UUID) (sqlc.User, error)
	ListUsers(context.Context) ([]sqlc.User, error)
}

// UserService provides business operations for users.
type UserService struct {
	store UserStore
}

// User is a user in the Waypoint domain.
type User struct {
	// ID is the user's UUID.
	ID uuid.UUID
	// Email is the user's email address.
	Email string
	// DisplayName is the name shown for the user.
	DisplayName string
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time
}

func NewUserService(store UserStore) *UserService {
	return &UserService{
		store: store,
	}
}

// CreateUser creates a user with the supplied email address and display name.
func (s *UserService) CreateUser(ctx context.Context, email string, displayName string) (*User, error) {
	user, err := s.store.CreateUser(ctx, sqlc.CreateUserParams{
		ID:          uuid.NewV7(),
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, AlreadyExistsError{
				What: "email",
			}
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return &User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Time,
	}, nil
}

// GetUser retrieves a user by UUID.
func (s *UserService) GetUser(ctx context.Context, ID uuid.UUID) (*User, error) {
	user, err := s.store.SelectUser(ctx, ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{
				What: "user",
			}
		}

		return nil, fmt.Errorf("get user: %w", err)
	}

	return &User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Time,
	}, nil
}

// ListUsers returns all users ordered by display name.
func (s *UserService) ListUsers(ctx context.Context) ([]User, error) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	var result []User
	for _, user := range users {
		result = append(result, User{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   user.CreatedAt.Time,
		})
	}

	return result, nil
}
