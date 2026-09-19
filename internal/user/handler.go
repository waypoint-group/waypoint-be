package user

import (
	"errors"
	"log"
	"net/http"

	"github.com/waypoint-group/waypoint-be/internal/httpx"
	"github.com/waypoint-group/waypoint-be/internal/middleware"
)

// Handler serves user profiles and registration over HTTP.
type Handler struct {
	service   *Service
	jwtConfig middleware.JWTConfig
}

// NewHandler constructs a user handler with access token verification.
func NewHandler(service *Service, jwtConfig middleware.JWTConfig) *Handler {
	return &Handler{service: service, jwtConfig: jwtConfig}
}

// CreateUser validates and creates a user from the request body.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	_, claims, err := middleware.ExtractAndValidateJWT(r, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request CreateUserRequest
	if err := httpx.ReadJSON(r, &request); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	u, err := h.service.CreateUser(r.Context(), CreateUserInput{
		Email:       request.Email,
		UserName:    request.UserName,
		DisplayName: request.DisplayName,
		Identity:    ExternalIdentity{Issuer: claims.Issuer, Subject: claims.Subject},
	})
	if err != nil {
		writeError(w, err)
		return
	}

	response := CreateUserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt,
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, response); err != nil {
		writeError(w, err)
	}
}

// GetUser retrieves the user identified by the request path.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ID, err := httpx.PathID(r, "id")
	if err != nil {
		http.Error(w, ErrInvalidId.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.service.ReadUser(r.Context(), ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
			return
		}

		writeError(w, err)
		return
	}

	response := GetUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt,
	}
	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		writeError(w, err)
	}
}

// ListUsers writes all users returned by the accounts service.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	response := ListUsersResponse{Users: make([]GetUserResponse, 0, len(users))}
	for _, user := range users {
		response.Users = append(response.Users, GetUserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
			CreatedAt:   user.CreatedAt,
		})
	}
	if err := httpx.WriteJSON(w, http.StatusOK, response); err != nil {
		writeError(w, err)
	}
}

// Me writes the local user profile associated with the bearer token's identity.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	_, claims, err := middleware.ExtractAndValidateJWT(r, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.service.ReadUserByIdentity(
		r.Context(),
		claims.Issuer,
		claims.Subject,
	)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, ErrNotFound.Error(), http.StatusNotFound)
		} else {
			writeError(w, err)
		}
		return
	}

	err = httpx.WriteJSON(
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
		writeError(w, err)
	}
}

// writeError maps domain sentinels to HTTP statuses and hides unexpected causes.
func writeError(w http.ResponseWriter, err error) {
	status, public := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, public = http.StatusNotFound, err.Error()
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrUserNameTaken), errors.Is(err, ErrIdentityTaken):
		status, public = http.StatusConflict, err.Error()
	case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidUserName), errors.Is(err, ErrInvalidDisplayName), errors.Is(err, ErrInvalidIdentity):
		status, public = http.StatusBadRequest, "invalid request body: "+err.Error()
	case errors.Is(err, ErrInvalidId):
		status, public = http.StatusBadRequest, err.Error()
	}
	if status == http.StatusInternalServerError {
		log.Printf("user error: %v", err)
	}
	http.Error(w, public, status)
}
