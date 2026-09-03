package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/service"
)

type CreateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type CreateUserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type GetUserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListUsersResponse struct {
	Users []GetUserResponse `json:"users"`
}

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

	user, err := h.services.Users.CreateUser(r.Context(), email, displayName)
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

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.services.Users.GetUser(r.Context(), ID)
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

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.services.Users.ListUsers(r.Context())
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
