//go:build integration

package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
	"github.com/waypoint-group/waypoint-be/internal/testlib"
)

func TestUsers_Create(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	createTestUser(t, uut, "ada@example.com", "Ada Lovelace")
}

func TestUsers_Get(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	created := createTestUser(t, uut, "ada@example.com", "Ada Lovelace")
	response, err := uut.Get("/users/" + created.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var user httpapi.GetUserResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode user: %v", err)
	}
	if user.ID != created.ID || user.Email != created.Email ||
		user.DisplayName != created.DisplayName || !user.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("expected user %+v, got %+v", created, user)
	}
}

func TestUsers_GetMissing(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	response, err := uut.Get("/users/" + uuid.New().String())
	if err != nil {
		t.Fatalf("failed to get missing user: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", response.StatusCode)
	}
}

func TestUsers_GetInvalidID(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	response, err := uut.Get("/users/not-a-uuid")
	if err != nil {
		t.Fatalf("failed to get user with invalid ID: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read error response: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != "invalid user id" {
		t.Errorf("expected error %q, got %q", "invalid user id", got)
	}
}

func TestUsers_List(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	assertUsers := func(want []httpapi.CreateUserResponse) {
		t.Helper()
		response, err := uut.Get("/users")
		if err != nil {
			t.Fatalf("failed to list users: %v", err)
		}
		defer func() {
			_ = response.Body.Close()
		}()

		if response.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", response.StatusCode)
		}
		var list httpapi.ListUsersResponse
		if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
			t.Fatalf("failed to decode users: %v", err)
		}
		if len(list.Users) != len(want) {
			t.Fatalf("expected %d users, got %d: %+v", len(want), len(list.Users), list.Users)
		}
		for i, user := range list.Users {
			if user.ID != want[i].ID || user.Email != want[i].Email ||
				user.DisplayName != want[i].DisplayName || !user.CreatedAt.Equal(want[i].CreatedAt) {
				t.Errorf("expected user at index %d to be %+v, got %+v", i, want[i], user)
			}
		}
	}

	assertUsers(nil)
	grace := createTestUser(t, uut, "grace@example.com", "Grace Hopper")
	ada := createTestUser(t, uut, "ada@example.com", "Ada Lovelace")
	assertUsers([]httpapi.CreateUserResponse{ada, grace})
}

func createTestUser(t *testing.T, uut *testlib.Waypoint, email, displayName string) httpapi.CreateUserResponse {
	t.Helper()
	body, err := json.Marshal(httpapi.CreateUserRequest{Email: email, DisplayName: displayName})
	if err != nil {
		t.Fatalf("failed to encode user: %v", err)
	}
	response, err := uut.Post("/users", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected JSON content type, got %q", got)
	}
	var user httpapi.CreateUserResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode created user: %v", err)
	}
	if _, err := uuid.Parse(user.ID); err != nil {
		t.Errorf("expected a valid user UUID, got %q: %v", user.ID, err)
	}
	if user.Email != email || user.DisplayName != displayName {
		t.Errorf("expected email %q and display name %q, got %+v", email, displayName, user)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected a nonzero creation timestamp")
	}
	return user
}
