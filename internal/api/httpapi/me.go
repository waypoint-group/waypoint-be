package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5/request"
	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

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

	_, claims, err := middleware.ValidateJWT(rawJWT, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.services.UserIdentities.GetUserByIdentity(
		r.Context(),
		claims.Subject,
		claims.Issuer,
	)
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
