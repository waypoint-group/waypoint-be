//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"uuid"

	userapi "github.com/waypoint-group/waypoint-be/internal/user"
	"github.com/waypoint-group/waypoint-be/tests/testlib"
)

func TestUsers_Create(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	t.Run("duplicate email", func(t *testing.T) {
		token, err := testEnv.Keycloak().AccessToken(t.Context(), "linked-ada", "test-password")
		if err != nil {
			t.Fatalf("log into Keycloak: %v", err)
		}

		response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
			strings.NewReader(`{"email":"ada@example.com","user_name":"another-user","display_name":"Another User"}`), token)
		if err != nil {
			t.Fatalf("register duplicate email: %v", err)
		}
		defer func() { _ = response.Body.Close() }()

		if response.StatusCode != http.StatusConflict {
			body, err := io.ReadAll(response.Body)
			if err == nil {
				t.Logf("response body: %s", string(body))
			}
			t.Fatalf("expected status 409, got %d", response.StatusCode)
		}

		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatalf("read conflict response: %v", err)
		}
		if got := strings.TrimSpace(string(body)); got != "email already in use" {
			t.Errorf("expected email conflict, got %q", got)
		}

		assertUsers(t, uut, []userapi.CreateUserResponse{created})
	})
}

func TestUsers_CreateDuplicateUserName(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada")
	token, err := testEnv.Keycloak().AccessToken(t.Context(), "linked-ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
		strings.NewReader(`{"email":"linked-ada@example.com","user_name":"ada","display_name":"Another Ada"}`), token)
	if err != nil {
		t.Fatalf("register duplicate user name: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if response.StatusCode != http.StatusConflict || strings.TrimSpace(string(body)) != "user name already in use" {
		t.Fatalf("expected user name conflict, got %d: %s", response.StatusCode, body)
	}

	assertUsers(t, uut, []userapi.CreateUserResponse{created})

	// A conflict must leave the second identity available for registration.
	createTestUser(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
}

func TestUsers_CreateDuplicateEmail(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada")
	token, err := testEnv.Keycloak().AccessToken(t.Context(), "linked-ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
		strings.NewReader(`{"email":"ada@example.com","user_name":"linked-ada","display_name":"Another Ada"}`), token)
	if err != nil {
		t.Fatalf("register duplicate email: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if response.StatusCode != http.StatusConflict || strings.TrimSpace(string(body)) != "email already in use" {
		t.Fatalf("expected email conflict, got %d: %s", response.StatusCode, body)
	}

	assertUsers(t, uut, []userapi.CreateUserResponse{created})

	// A conflict must leave the second identity available for registration.
	createTestUser(t, uut, "linked-ada@example.com", "linked-ada", "Another Ada")
}

func TestUsers_CreateDuplicateIdentity(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	token, err := testEnv.Keycloak().AccessToken(t.Context(), "ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users",
		strings.NewReader(`{"email":"another@example.com","user_name":"another-profile","display_name":"Another Profile"}`), token)
	if err != nil {
		t.Fatalf("repeat registration: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusConflict {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status 409, got %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read conflict response: %v", err)
	}
	if got := strings.TrimSpace(string(body)); got != "user identity already in use" {
		t.Errorf("expected identity conflict, got %q", got)
	}

	// The failed registration must roll back the second profile.
	assertUsers(t, uut, []userapi.CreateUserResponse{created})

	meResponse, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
	if err != nil {
		t.Fatalf("get original profile: %v", err)
	}
	defer func() { _ = meResponse.Body.Close() }()

	if meResponse.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status 200, got %d", meResponse.StatusCode)
	}

	var profile userapi.MeResponse
	if err := json.NewDecoder(meResponse.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	if profile.User.ID != created.ID {
		t.Errorf("expected original profile %s, got %s", created.ID, profile.User.ID)
	}
}

func TestUsers_Get(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	response, err := uut.Request(t.Context(), http.MethodGet, "/users/"+created.ID, nil)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var get userapi.GetUserResponse
	if err := json.NewDecoder(response.Body).Decode(&get); err != nil {
		t.Fatalf("failed to decode user: %v", err)
	}

	same := get.ID == created.ID &&
		get.Email == created.Email &&
		get.UserName == created.UserName &&
		get.DisplayName == created.DisplayName &&
		get.CreatedAt.Equal(created.CreatedAt)
	if !same {
		t.Errorf("expected user %+v, got %+v", created, get)
	}
}

func TestUsers_GetMissing(t *testing.T) {
	uut := newWaypoint(t)

	response, err := uut.Request(t.Context(), http.MethodGet, "/users/"+uuid.New().String(), nil)
	if err != nil {
		t.Fatalf("failed to get missing user: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Errorf("expected status 404, got %d", response.StatusCode)
	}
}

func TestUsers_GetInvalidID(t *testing.T) {
	uut := newWaypoint(t)

	response, err := uut.Request(t.Context(), http.MethodGet, "/users/not-a-uuid", nil)
	if err != nil {
		t.Fatalf("failed to get user with invalid ID: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusBadRequest {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
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
	uut := newWaypoint(t)

	assertUsers(t, uut, nil)
	ada := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	linkedAda := createTestUser(t, uut, "linked-ada@example.com", "linked-ada", "Ada Lovelace")
	assertUsers(t, uut, []userapi.CreateUserResponse{ada, linkedAda})
}

func createTestUser(t *testing.T, uut *testlib.TestWaypoint, email, userName, displayName string) userapi.CreateUserResponse {
	t.Helper()
	body, err := json.Marshal(userapi.CreateUserRequest{Email: email, UserName: userName, DisplayName: displayName})
	if err != nil {
		t.Fatalf("failed to encode user: %v", err)
	}

	token, err := testEnv.Keycloak().AccessToken(t.Context(), userName, "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users", bytes.NewReader(body), token)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("expected JSON content type, got %q", got)
	}

	var user userapi.CreateUserResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatalf("failed to decode created user: %v", err)
	}
	if _, err := uuid.Parse(user.ID); err != nil {
		t.Errorf("expected a valid user UUID, got %q: %v", user.ID, err)
	}
	if user.Email != email || user.UserName != userName || user.DisplayName != displayName {
		t.Errorf("expected email %q, user name %q, and display name %q, got %+v", email, userName, displayName, user)
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected a nonzero creation timestamp")
	}

	return user
}

func assertUsers(t *testing.T, uut *testlib.TestWaypoint, want []userapi.CreateUserResponse) {
	t.Helper()
	response, err := uut.Request(t.Context(), http.MethodGet, "/users", nil)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var list userapi.ListUsersResponse
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		t.Fatalf("failed to decode users: %v", err)
	}

	if len(list.Users) != len(want) {
		t.Fatalf("expected %d users, got %d: %+v", len(want), len(list.Users), list.Users)
	}

	for i, user := range list.Users {
		same := user.ID == want[i].ID &&
			user.Email == want[i].Email &&
			user.UserName == want[i].UserName &&
			user.DisplayName == want[i].DisplayName &&
			user.CreatedAt.Equal(want[i].CreatedAt)
		if !same {
			t.Errorf("expected user at index %d to be %+v, got %+v", i, want[i], user)
		}
	}
}

func TestUsers_CreateValidation(t *testing.T) {
	uut := newWaypoint(t)

	token, err := testEnv.Keycloak().AccessToken(t.Context(), "ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"missing email", `{"user_name":"ada","display_name":"Ada"}`, "invalid request body: invalid email"},
		{"blank email", `{"email":"  ","user_name":"ada","display_name":"Ada"}`, "invalid request body: invalid email"},
		{"blank user name", `{"email":"ada@example.com","user_name":"  ","display_name":"Ada"}`, "invalid request body: invalid user name"},
		{"blank display name", `{"email":"ada@example.com","user_name":"ada","display_name":"  "}`, "invalid request body: invalid display name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := uut.RequestWithAccessToken(t.Context(), http.MethodPost, "/users", strings.NewReader(tc.body), token)
			if err != nil {
				t.Fatalf("register invalid user: %v", err)
			}
			defer func() { _ = response.Body.Close() }()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", response.StatusCode, body)
			}
			if got := strings.TrimSpace(string(body)); got != tc.want {
				t.Errorf("unexpected validation error: %q", got)
			}

			assertUsers(t, uut, nil)
		})
	}

	// Rejected input must not consume the external identity.
	createTestUser(t, uut, "ada@example.com", "ada", "Ada")
}
