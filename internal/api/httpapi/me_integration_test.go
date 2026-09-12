//go:build integration

package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/testlib"
)

func TestMe(t *testing.T) {
	uut, err := testlib.NewWaypoint()
	if err != nil {
		t.Fatalf("start waypoint: %v", err)
	}
	t.Cleanup(uut.Close)

	user, err := uut.Services().Users.Create(t.Context(), "ada@example.com", "Ada Lovelace")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	for _, tc := range []struct {
		name        string
		username    string
		linked      bool
		otherIssuer bool
		status      int
	}{
		{"valid", "ada", true, false, http.StatusOK},
		{"linked account", "linked-ada", true, false, http.StatusOK},
		{"unknown identity", "missing", false, false, http.StatusNotFound},
		{"identity belongs to another issuer", "other-realm-only", true, true, http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := uut.Keycloak().AccessToken(t.Context(), tc.username, "test-password")
			if err != nil {
				t.Fatalf("log into Keycloak: %v", err)
			}

			_, claims, err := middleware.ValidateJWT(token, uut.Keycloak().JWTConfig)
			if err != nil {
				t.Fatalf("verify Keycloak access token: %v", err)
			}

			if tc.linked {
				issuer := claims.Issuer
				if tc.otherIssuer {
					issuer = "https://other.example.com"
				}

				_, err := uut.Services().UserIdentities.Create(
					t.Context(),
					user.ID,
					claims.Subject,
					issuer,
				)
				if err != nil {
					t.Fatalf("create identity: %v", err)
				}
			}

			response, err := uut.RequestWithAccessToken(t.Context(), http.MethodGet, "/me", nil, token)
			if err != nil {
				t.Fatalf("request /me: %v", err)
			}
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Fatalf("close response body: %v", err)
				}
			}()
			if response.StatusCode != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, response.StatusCode)
			}
			if tc.status == http.StatusOK {
				var got httpapi.MeResponse
				if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
					t.Fatalf("decode profile: %v", err)
				}
				if got.ID != user.ID.String() || got.Email != user.Email || got.DisplayName != user.DisplayName || !got.CreatedAt.Equal(user.CreatedAt) {
					t.Errorf("expected profile %+v, got %+v", user, got)
				}
			}
		})
	}

	t.Run("missing authorization", func(t *testing.T) {
		response, err := uut.Request(t.Context(), http.MethodGet, "/me", nil)
		if err != nil {
			t.Fatalf("request /me: %v", err)
		}
		defer func() {
			if err := response.Body.Close(); err != nil {
				t.Fatalf("close response body: %v", err)
			}
		}()
		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", response.StatusCode)
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		_, err := uut.Keycloak().AccessToken(t.Context(), "ada", "wrong-password")
		if err == nil {
			t.Fatal("expected Keycloak to reject incorrect password")
		}
	})
}
