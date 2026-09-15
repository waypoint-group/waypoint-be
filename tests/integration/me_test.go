//go:build integration

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
)

func TestMe(t *testing.T) {
	uut := newWaypoint(t)

	created := createTestUser(t, uut, "ada@example.com", "ada", "Ada Lovelace")

	for _, tc := range []struct {
		name     string
		username string
		status   int
	}{
		{"valid", "ada", http.StatusOK},
		{"unknown identity", "missing", http.StatusNotFound},
		// TODO: Linking accounts not implemented yet.
		//
		// {"linked account", "linked-ada", "requires an HTTP endpoint to link another identity to the registered user", http.StatusOK},
		// {"identity belongs to another issuer", "other-realm-only", "requires HTTP setup for an identity linked under another issuer", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := testEnv.Keycloak().AccessToken(t.Context(), tc.username, "test-password")
			if err != nil {
				t.Fatalf("log into Keycloak: %v", err)
			}

			response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
			if err != nil {
				t.Fatalf("request /me: %v", err)
			}
			defer func() { _ = response.Body.Close() }()

			if response.StatusCode != tc.status {
				body, err := io.ReadAll(response.Body)
				if err == nil {
					t.Logf("response body: %s", string(body))
				}
				t.Fatalf("expected status %d, got %d", tc.status, response.StatusCode)
			}

			if tc.status == http.StatusOK {
				var me httpapi.MeResponse
				if err := json.NewDecoder(response.Body).Decode(&me); err != nil {
					t.Fatalf("decode profile: %v", err)
				}

				same := me.ID == created.ID &&
					me.Email == created.Email &&
					me.UserName == created.UserName &&
					me.DisplayName == created.DisplayName &&
					me.CreatedAt.Equal(created.CreatedAt)
				if !same {
					t.Errorf("expected profile %+v, got %+v", created, me)
				}
			}
		})
	}

	t.Run("missing authorization", func(t *testing.T) {
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
	})

	t.Run("incorrect password", func(t *testing.T) {
		_, err := testEnv.Keycloak().AccessToken(t.Context(), "ada", "wrong-password")
		if err == nil {
			t.Fatal("expected Keycloak to reject incorrect password")
		}
	})
}
