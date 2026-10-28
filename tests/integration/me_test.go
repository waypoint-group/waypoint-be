//go:build integration

package integration

import (
	"encoding/json"
	"github.com/waypoint-group/waypoint-be/internal/api"
	"io"
	"net/http"
	"testing"
)

func TestMe_Valid(t *testing.T) {
	uut := newWaypoint(t)
	created := createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	token, err := testEnv.Keycloak().AccessToken(t.Context(), "ada", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
	if err != nil {
		t.Fatalf("request /me: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var me api.MeResponse
	if err := json.NewDecoder(response.Body).Decode(&me); err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	same := me.User.ID == created.ID &&
		me.User.Email == created.Email &&
		me.User.UserName == created.UserName &&
		me.User.DisplayName == created.DisplayName &&
		me.User.CreatedAt.Equal(created.CreatedAt)
	if !same {
		t.Errorf("expected profile %+v, got %+v", created, me)
	}
}

func TestMe_UnknownIdentity(t *testing.T) {
	uut := newWaypoint(t)
	createTestUserChecked(t, uut, "ada@example.com", "ada", "Ada Lovelace")
	token, err := testEnv.Keycloak().AccessToken(t.Context(), "missing", "test-password")
	if err != nil {
		t.Fatalf("log into Keycloak: %v", err)
	}

	response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
	if err != nil {
		t.Fatalf("request /me: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusNotFound {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.StatusCode)
	}

}

func TestMe_MissingAuthorization(t *testing.T) {
	uut := newWaypoint(t)
	response, err := uut.Request(t.Context(), http.MethodGet, "/me", nil)
	if err != nil {
		t.Fatalf("request /me: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusUnauthorized {
		body, err := io.ReadAll(response.Body)
		if err == nil {
			t.Logf("response body: %s", string(body))
		}
		t.Errorf("expected status 401, got %d", response.StatusCode)
	}
}
