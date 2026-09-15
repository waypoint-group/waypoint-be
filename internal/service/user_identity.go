package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/db"
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
	// Issuer identifies the authority that issued the subject.
	Issuer string
	// Subject identifies the external account within its issuer.
	Subject string
	// CreatedAt is the time at which the identity was linked.
	CreatedAt time.Time
}

// NewUserIdentityService constructs a user identity service backed by the supplied store.
func NewUserIdentityService(store UserIdentityStore) *UserIdentityService {
	return &UserIdentityService{
		store: store,
	}
}

// Create links an authentication identity to a local user.
func (s *UserIdentityService) Create(ctx context.Context, userId uuid.UUID, subject string, issuer string) (*UserIdentity, error) {
	user, err := s.store.CreateUserIdentity(ctx, sqlc.CreateUserIdentityParams{
		ID:      uuid.NewV7(),
		UserID:  userId,
		Issuer:  issuer,
		Subject: subject,
	})
	if err != nil {
		if db.IsUniqueViolation(err, "user_identities_identity_unique") {
			return nil, AlreadyExistsError{What: "user identity"}
		}
		return nil, fmt.Errorf("create user identity: %w", err)
	}

	return &UserIdentity{
		ID:        user.ID,
		UserID:    user.UserID,
		Issuer:    user.Issuer,
		Subject:   user.Subject,
		CreatedAt: user.CreatedAt.Time,
	}, nil
}

// Get retrieves an identity mapping by UUID.
func (s *UserIdentityService) Get(ctx context.Context, id uuid.UUID) (*UserIdentity, error) {
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
		ID:        userIdentity.ID,
		UserID:    userIdentity.UserID,
		Issuer:    userIdentity.Issuer,
		Subject:   userIdentity.Subject,
		CreatedAt: userIdentity.CreatedAt.Time,
	}, nil
}

// List returns all identity mappings.
func (s *UserIdentityService) List(ctx context.Context) ([]UserIdentity, error) {
	userIdentities, err := s.store.ListUserIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user identities: %w", err)
	}

	result := make([]UserIdentity, 0, len(userIdentities))
	for _, userIdentity := range userIdentities {
		result = append(result, UserIdentity{
			ID:        userIdentity.ID,
			UserID:    userIdentity.UserID,
			Issuer:    userIdentity.Issuer,
			Subject:   userIdentity.Subject,
			CreatedAt: userIdentity.CreatedAt.Time,
		})
	}

	return result, nil
}

// GetUserByIdentity retrieves the local user matching both subject and issuer.
func (s *UserIdentityService) GetUserByIdentity(ctx context.Context, issuer string, subject string) (*User, error) {
	user, err := s.store.SelectUserByIdentity(ctx, sqlc.SelectUserByIdentityParams{
		Issuer:  issuer,
		Subject: subject,
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
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Time,
	}, nil
}
