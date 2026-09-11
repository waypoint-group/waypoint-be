package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/service"
)

// CreateUserRequest contains the fields accepted when creating a user.
type CreateUserRequest struct {
	// Email is the user's email address.
	Email string `json:"email"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
}

// CreateUserResponse is returned after a user is created.
type CreateUserResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// GetUserResponse is returned when retrieving a user.
type GetUserResponse struct {
	// ID is the user's UUID in its canonical string form.
	ID string `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
	// CreatedAt is the time at which the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// ListUsersResponse contains the users returned by the list endpoint.
type ListUsersResponse struct {
	// Users contains the users in the response.
	Users []GetUserResponse `json:"users"`
}

// CreateUser validates and creates a user from the request body.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeInvalidRequestBody(w, err)
		return
	}

	email := strings.TrimSpace(request.Email)
	displayName := strings.TrimSpace(request.DisplayName)

	if email == "" || displayName == "" {
		writeInvalidRequestBody(w, errors.New("email and display name are required"))
		return
	}

	user, err := h.services.Users.Create(r.Context(), email, displayName)
	if err != nil {
		var alreadyExists service.AlreadyExistsError
		if errors.As(err, &alreadyExists) {
			http.Error(w, alreadyExists.Error(), http.StatusConflict)
			return
		}

		writeInternalServerError(w, err)
		return
	}

	response := CreateUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt,
	}
	if err := writeJSON(w, http.StatusCreated, response); err != nil {
		writeInternalServerError(w, err)
	}
}

// GetUser retrieves the user identified by the request path.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.services.Users.Get(r.Context(), ID)
	if err != nil {
		var notFound service.NotFoundError
		if errors.As(err, &notFound) {
			http.Error(w, notFound.Error(), http.StatusNotFound)
			return
		}

		writeInternalServerError(w, err)
		return
	}

	response := GetUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt,
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		writeInternalServerError(w, err)
	}
}

// ListUsers writes all users returned by the user service.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.services.Users.List(r.Context())
	if err != nil {
		writeInternalServerError(w, err)
		return
	}

	response := ListUsersResponse{}
	for _, user := range users {
		response.Users = append(response.Users, GetUserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			DisplayName: user.DisplayName,
			CreatedAt:   user.CreatedAt,
		})
	}
	if err := writeJSON(w, http.StatusOK, response); err != nil {
		writeInternalServerError(w, err)
	}
}
