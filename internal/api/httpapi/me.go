package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

// JWTConfig contains the trusted issuer, audience, and key resolver for access tokens.
// KeyFunc must return trusted RSA public keys; it must not trust keys supplied by the token.
type JWTConfig struct {
	// Issuer is the exact issuer URL accepted by the API.
	Issuer string
	// Audience is the audience required in access tokens for this API.
	Audience string
	// KeyFunc resolves the RSA public key used to verify a token's signature.
	KeyFunc jwt.Keyfunc
}

// Option configures an HTTP API handler.
type Option func(*Handler)

// WithJWTVerification configures RS256 access token verification for authenticated routes.
// Missing issuer, audience, or key resolver causes authentication to fail closed.
func WithJWTVerification(config JWTConfig) Option {
	return func(h *Handler) { h.jwtConfig = config }
}

// MeResponse contains the authenticated user's profile.
type MeResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// Me writes the local user profile associated with the bearer token's identity.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// Forbid duplicate authorization headers.
	headers := r.Header.Values("Authorization")
	if len(headers) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	rawJWT, err := (request.BearerExtractor{}).ExtractToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.jwtConfig.KeyFunc == nil || h.jwtConfig.Issuer == "" || h.jwtConfig.Audience == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(
		rawJWT,
		&claims,
		h.jwtConfig.KeyFunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(h.jwtConfig.Issuer),
		jwt.WithAudience(h.jwtConfig.Audience),
	)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Subject == "" || claims.Issuer == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.services.UserIdentities.GetUserByIdentity(r.Context(), claims.Subject, claims.Issuer)
	if err != nil {
		var notFound service.NotFoundError
		if errors.As(err, &notFound) {
			http.Error(w, "Not Found", http.StatusNotFound)
		} else {
			writeInternalServerError(w, err)
		}
		return
	}

	err = writeJSON(
		w,
		http.StatusOK,
		MeResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   user.CreatedAt,
		},
	)
	if err != nil {
		writeInternalServerError(w, err)
	}
}
