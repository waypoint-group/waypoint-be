//go:build integration

package httpapi_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/waypoint-group/waypoint-be/internal/api/httpapi"
	"github.com/waypoint-group/waypoint-be/internal/testlib"
	"github.com/waypoint-group/waypoint-be/internal/waypoint"
)

// TestMe exercises signed access tokens against the HTTP routes and a migrated PostgreSQL database.
func TestMe(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate untrusted key: %v", err)
	}
	const issuer = "https://auth.example.com/realms/waypoint"
	const audience = "waypoint-api"
	postgres, err := testlib.NewPostgres()
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := postgres.Terminate(ctx); err != nil {
			t.Errorf("stop PostgreSQL: %v", err)
		}
	})
	app, err := waypoint.New(&waypoint.Config{
		DatabaseURL: postgres.URL, Migrate: true,
		JWT: httpapi.JWTConfig{
			Issuer:   issuer,
			Audience: audience,
			KeyFunc:  func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
		},
	})
	if err != nil {
		t.Fatalf("start waypoint: %v", err)
	}
	t.Cleanup(app.Database.Close)

	server := httptest.NewServer(app.Handler)
	t.Cleanup(server.Close)
	user, err := app.Services.Users.CreateUser(t.Context(), "ada@example.com", "Ada Lovelace")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	for _, subject := range []string{"ada", "linked-ada"} {
		if _, err := app.Services.UserIdentities.CreateUserIdentity(t.Context(), user.ID, subject, issuer); err != nil {
			t.Fatalf("create identity: %v", err)
		}
	}

	// A matching subject from a different issuer must not resolve to this user.
	if _, err := app.Services.UserIdentities.CreateUserIdentity(t.Context(), user.ID, "other-realm-only", "https://other.example.com"); err != nil {
		t.Fatalf("create other issuer identity: %v", err)
	}

	claims := func() jwt.RegisteredClaims {
		return jwt.RegisteredClaims{
			Subject:   "ada",
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
	}
	sign := func(c jwt.RegisteredClaims, method jwt.SigningMethod, signingKey any) string {
		t.Helper()
		token, err := jwt.NewWithClaims(method, c).SignedString(signingKey)
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return token
	}
	request := func(t *testing.T, headers []string, wantStatus int) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/me", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		for _, header := range headers {
			req.Header.Add("Authorization", header)
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatalf("request /me: %v", err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != wantStatus {
			t.Fatalf("expected status %d, got %d", wantStatus, response.StatusCode)
		}
		if wantStatus != http.StatusOK {
			return
		}
		if got := response.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}
		var got httpapi.MeResponse
		if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
			t.Fatalf("decode profile: %v", err)
		}
		if got.ID != user.ID.String() || got.Email != user.Email || got.DisplayName != user.DisplayName || !got.CreatedAt.Equal(user.CreatedAt) {
			t.Errorf("expected profile %+v, got %+v", user, got)
		}
	}

	valid := sign(claims(), jwt.SigningMethodRS256, key)

	for _, tc := range []struct {
		name    string
		headers []string
		status  int
	}{
		{"valid", []string{"Bearer " + valid}, http.StatusOK},
		{"missing header", nil, http.StatusUnauthorized},
		{"wrong scheme", []string{"Basic " + valid}, http.StatusUnauthorized},
		{"malformed JWT", []string{"Bearer invalid"}, http.StatusUnauthorized},
		{"untrusted signature", []string{"Bearer " + sign(claims(), jwt.SigningMethodRS256, otherKey)}, http.StatusUnauthorized},
		{"wrong algorithm", []string{"Bearer " + sign(claims(), jwt.SigningMethodHS256, []byte("test-secret"))}, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) { request(t, tc.headers, tc.status) })
	}
	for _, tc := range []struct {
		name   string
		change func(*jwt.RegisteredClaims)
		status int
	}{
		{"linked account", func(c *jwt.RegisteredClaims) { c.Subject = "linked-ada" }, http.StatusOK},
		{"unknown identity", func(c *jwt.RegisteredClaims) { c.Subject = "missing" }, http.StatusNotFound},
		{"identity belongs to another issuer", func(c *jwt.RegisteredClaims) { c.Subject = "other-realm-only" }, http.StatusNotFound},
		{"missing subject", func(c *jwt.RegisteredClaims) { c.Subject = "" }, http.StatusUnauthorized},
		{"missing issuer", func(c *jwt.RegisteredClaims) { c.Issuer = "" }, http.StatusUnauthorized},
		{"wrong issuer", func(c *jwt.RegisteredClaims) { c.Issuer = "https://other.example.com" }, http.StatusUnauthorized},
		{"missing audience", func(c *jwt.RegisteredClaims) { c.Audience = nil }, http.StatusUnauthorized},
		{"wrong audience", func(c *jwt.RegisteredClaims) { c.Audience = jwt.ClaimStrings{"another-api"} }, http.StatusUnauthorized},
		{"missing expiration", func(c *jwt.RegisteredClaims) { c.ExpiresAt = nil }, http.StatusUnauthorized},
		{"expired", func(c *jwt.RegisteredClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }, http.StatusUnauthorized},
		{"not yet valid", func(c *jwt.RegisteredClaims) { c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour)) }, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := claims()
			tc.change(&c)
			request(t, []string{"Bearer " + sign(c, jwt.SigningMethodRS256, key)}, tc.status)
		})
	}
	t.Run("verification not configured", func(t *testing.T) {
		handler := httpapi.New(app.Database, app.Services).Routes()
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", response.Code)
		}
	})
	t.Run("database failure", func(t *testing.T) {
		app.Database.Close()
		request(t, []string{"Bearer " + valid}, http.StatusInternalServerError)
	})
}
