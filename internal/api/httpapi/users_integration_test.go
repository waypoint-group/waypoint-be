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

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	t.Run("duplicate email", func(t *testing.T) {
		token, err := uut.Keycloak().AccessToken(t.Context(), "linked-ada", "test-password")
		if err != nil {
			t.Fatalf("log into Keycloak: %v", err)
		}
		response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
			strings.NewReader(`{"email":"ada@example.com","display_name":"Another User"}`), token)
		if err != nil {
			t.Fatalf("register duplicate email: %v", err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("expected status 409, got %d", response.StatusCode)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read conflict response: %v", err)
		}
		if got := strings.TrimSpace(string(body)); got != "email already exists" {
			t.Errorf("expected email conflict, got %q", got)
		}
		assertUsers(t, uut, []httpapi.CreateUserResponse{created})
	})
}

func TestUsers_CreateDuplicateIdentity(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	t.Cleanup(uut.Close)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	token, err := uut.Keycloak().AccessToken(t.Context(), "ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}
	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
		strings.NewReader(`{"email":"another@example.com","display_name":"Another Profile"}`), token)
	if err != nil {
		t.Fatalf("repeat registration: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read conflict response: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != "user identity already exists" {
		t.Errorf("expected identity conflict, got %q", got)
	}
	// The failed registration must roll back the second profile.
	assertUsers(t, uut, []httpapi.CreateUserResponse{created})

	meResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
	if err != nil {
		t.Fatalf("get original profile: %v", err)
	}
	defer func() { _ = meResponse.Body.Close() }()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", meResponse.StatusCode)
	}
	var profile httpapi.MeResponse
	if err := json.NewDecoder(meResponse.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if profile.ID != created.ID {
		t.Errorf("expected original profile %s, got %s", created.ID, profile.ID)
	}
}

func TestUsers_Get(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("failed to run waypoint: %v", err)
	}
	defer uut.Close()

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	response, err := uut.Request(t.Context(), http.MethodGet, "/users/"+created.ID, nil)
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

	response, err := uut.Request(t.Context(), http.MethodGet, "/users/"+uuid.New().String(), nil)
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

	response, err := uut.Request(t.Context(), http.MethodGet, "/users/not-a-uuid", nil)
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

	assertUsers(t, uut, nil)
	ada := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	linkedAda := createTestUser(t, uut, "linked-ada@example.com", "linked-ada", "Ada Lovelace")
	assertUsers(t, uut, []httpapi.CreateUserResponse{ada, linkedAda})
}

func createTestUser(t *testing.T, uut *testlib.Waypoint, email, userName, displayName string) httpapi.CreateUserResponse {
	t.Helper()
	body, err := json.Marshal(httpapi.CreateUserRequest{Email: email, DisplayName: displayName})
	if err != nil {
		t.Fatalf("failed to encode user: %v", err)
	}

	token, err := uut.Keycloak().AccessToken(t.Context(), userName, "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users", bytes.NewReader(body), token)
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

func assertUsers(t *testing.T, uut *testlib.Waypoint, want []httpapi.CreateUserResponse) {
	t.Helper()
	response, err := uut.Request(t.Context(), http.MethodGet, "/users", nil)
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
