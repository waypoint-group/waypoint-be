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

// UserIdentityStore defines the data access interface for user identity operations.
type UserIdentityStore interface {
	CreateUserIdentity(context.Context, sqlc.CreateUserIdentityParams) (sqlc.UserIdentity, error)
	SelectUserIdentity(context.Context, uuid.UUID) (sqlc.UserIdentity, error)
	SelectUserByIdentity(context.Context, sqlc.SelectUserByIdentityParams) (sqlc.User, error)
	ListUserIdentities(context.Context) ([]sqlc.UserIdentity, error)
}

// UserIdentityService provides business operations for user identities.
type UserIdentityService struct {
	store UserIdentityStore
}

// UserIdentity links an authentication identity to a local user.
type UserIdentity struct {
	// ID is the identity mapping's UUID.
	ID uuid.UUID
	// UserID is the UUID of the local user who owns this identity.
	UserID uuid.UUID
	// AuthSubject identifies the external account within its issuer.
	AuthSubject string
	// AuthIssuer identifies the authority that issued the subject.
	AuthIssuer string
	// CreatedAt is the time at which the identity was linked.
	CreatedAt time.Time
}

// NewUserIdentityService constructs a user identity service backed by the supplied store.
func NewUserIdentityService(store UserIdentityStore) *UserIdentityService {
	return &UserIdentityService{
		store: store,
	}
}

// CreateUserIdentity links an authentication identity to a local user.
func (s *UserIdentityService) CreateUserIdentity(ctx context.Context, userId uuid.UUID, authSubject string, authIssuer string) (*UserIdentity, error) {
	user, err := s.store.CreateUserIdentity(ctx, sqlc.CreateUserIdentityParams{
		ID:          uuid.NewV7(),
		UserID:      userId,
		AuthSubject: authSubject,
		AuthIssuer:  authIssuer,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, AlreadyExistsError{What: "user identity"}
		}
		return nil, fmt.Errorf("create user identity: %w", err)
	}

	return &UserIdentity{
		ID:          user.ID,
		UserID:      user.UserID,
		AuthSubject: user.AuthSubject,
		AuthIssuer:  user.AuthIssuer,
		CreatedAt:   user.CreatedAt.Time,
	}, nil
}

// GetUserIdentity retrieves an identity mapping by UUID.
func (s *UserIdentityService) GetUserIdentity(ctx context.Context, id uuid.UUID) (*UserIdentity, error) {
	userIdentity, err := s.store.SelectUserIdentity(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{
				What: "user identity",
			}
		}

		return nil, fmt.Errorf("get user identity: %w", err)
	}

	return &UserIdentity{
		ID:          userIdentity.ID,
		UserID:      userIdentity.UserID,
		AuthSubject: userIdentity.AuthSubject,
		AuthIssuer:  userIdentity.AuthIssuer,
		CreatedAt:   userIdentity.CreatedAt.Time,
	}, nil
}

// ListUserIdentities returns all identity mappings.
func (s *UserIdentityService) ListUserIdentities(ctx context.Context) ([]UserIdentity, error) {
	userIdentities, err := s.store.ListUserIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user identities: %w", err)
	}

	result := make([]UserIdentity, 0, len(userIdentities))
	for _, userIdentity := range userIdentities {
		result = append(result, UserIdentity{
			ID:          userIdentity.ID,
			UserID:      userIdentity.UserID,
			AuthSubject: userIdentity.AuthSubject,
			AuthIssuer:  userIdentity.AuthIssuer,
			CreatedAt:   userIdentity.CreatedAt.Time,
		})
	}

	return result, nil
}

// GetUserByIdentity retrieves the local user matching both subject and issuer.
func (s *UserIdentityService) GetUserByIdentity(ctx context.Context, authSubject string, authIssuer string) (*User, error) {
	user, err := s.store.SelectUserByIdentity(ctx, sqlc.SelectUserByIdentityParams{
		AuthSubject: authSubject,
		AuthIssuer:  authIssuer,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NotFoundError{
				What: "user",
			}
		}

		return nil, fmt.Errorf("get user by identity: %w", err)
	}

	return &User{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Time,
	}, nil
}
