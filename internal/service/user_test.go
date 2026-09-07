package service

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

type userStoreStub struct {
	users []sqlc.User
}

func (s *userStoreStub) CreateUser(_ context.Context, params sqlc.CreateUserParams) (sqlc.User, error) {
	user := sqlc.User{
		ID:          params.ID,
		Email:       params.Email,
		DisplayName: params.DisplayName,
		CreatedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	s.users = append(s.users, user)
	return user, nil
}

func (s *userStoreStub) SelectUser(_ context.Context, id uuid.UUID) (sqlc.User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}

	return sqlc.User{}, pgx.ErrNoRows
}

func (s *userStoreStub) ListUsers(context.Context) ([]sqlc.User, error) {
	return s.users, nil
}

func TestUserServiceCreatedAndSelectedUserMatch(t *testing.T) {
	store := &userStoreStub{}
	service := NewUserService(store)
	createdUser, err := service.CreateUser(context.Background(), "ada@example.com", "Ada Lovelace")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	selectedUser, err := service.GetUser(context.Background(), createdUser.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	usersMatch := createdUser.ID == selectedUser.ID &&
		createdUser.Email == selectedUser.Email &&
		createdUser.DisplayName == selectedUser.DisplayName &&
		createdUser.CreatedAt.Unix() == selectedUser.CreatedAt.Unix()
	if !usersMatch {
		t.Fatalf("expected %+v, got %+v", createdUser, selectedUser)
	}

	now := time.Now()
	if createdUser.CreatedAt.After(now) {
		t.Fatalf("unexpected timestamp: %+v", createdUser.CreatedAt)
	}
}

func TestUserServiceSelectNonExistentUserReturnsError(t *testing.T) {
	store := &userStoreStub{}
	service := NewUserService(store)

	_, err := service.GetUser(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserServiceListUserReturnsAllCreatedUsers(t *testing.T) {
	store := &userStoreStub{}
	service := NewUserService(store)

	user1, err := service.CreateUser(context.Background(), "user1@example.com", "User One")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	user2, err := service.CreateUser(context.Background(), "user2@example.com", "User Two")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	users, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	expectedUsers := []User{*user1, *user2}
	if !slices.Equal(users, expectedUsers) {
		t.Fatalf("expected %+v, got %+v", expectedUsers, users)
	}
}
