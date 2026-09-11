package service_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
	"github.com/waypoint-group/waypoint-be/internal/service"
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

func TestUser_CreateAndSelect(t *testing.T) {
	uut := service.NewUserService(&userStoreStub{})

	createdUser, err := uut.Create(t.Context(), "ada@example.com", "Ada Lovelace")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	selectedUser, err := uut.Get(t.Context(), createdUser.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
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

func TestUser_SelectMissing(t *testing.T) {
	uut := service.NewUserService(&userStoreStub{})

	_, err := uut.Get(t.Context(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUser_List(t *testing.T) {
	uut := service.NewUserService(&userStoreStub{})

	user1, err := uut.Create(t.Context(), "user1@example.com", "User One")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	user2, err := uut.Create(t.Context(), "user2@example.com", "User Two")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	users, err := uut.List(t.Context())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	expectedUsers := []service.User{*user1, *user2}
	if !slices.Equal(users, expectedUsers) {
		t.Fatalf("expected %+v, got %+v", expectedUsers, users)
	}
}
