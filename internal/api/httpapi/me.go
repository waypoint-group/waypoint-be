package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/services"
)

// MeResponse contains the authenticated user's profile.
type MeResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// Me writes the local user profile associated with the bearer token's identity.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	_, claims, err := middleware.ExtractAndValidateJWT(r, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.services.Accounts.ReadUserByIdentity(
		r.Context(),
		claims.Issuer,
		claims.Subject,
	)
	if err != nil {
		var notFound services.NotFoundError
		if errors.As(err, &notFound) {
			http.Error(w, "Not Found", http.StatusNotFound)
		} else {
			writeInternalServerError(w, err)
		}
		return
	}

	err = writeJSONResponse(
		w,
		http.StatusOK,
		MeResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
			CreatedAt:   user.CreatedAt,
		},
	)
	if err != nil {
		writeInternalServerError(w, err)
	}
}
