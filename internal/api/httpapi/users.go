package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/api/middleware"
	"github.com/waypoint-group/waypoint-be/internal/service"
)

// CreateUserRequest contains the fields accepted when creating a user.
type CreateUserRequest struct {
	// Email is the user's email address.
	Email string `json:"email"`
	// UserName is the user's handle.
	UserName string `json:"user_name"`
	// DisplayName is the name shown for the user.
	DisplayName string `json:"display_name"`
}

// CreateUserResponse is returned after a user is created.
type CreateUserResponse struct {
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

// GetUserResponse is returned when retrieving a user.
type GetUserResponse struct {
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

// ListUsersResponse contains the users returned by the list endpoint.
type ListUsersResponse struct {
	// Users contains the users in the response.
	Users []GetUserResponse `json:"users"`
}

// CreateUser validates and creates a user from the request body.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	_, claims, err := middleware.ExtractAndValidateJWT(r, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request CreateUserRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		writeInvalidRequestBody(w, err)
		return
	}

	email := strings.TrimSpace(request.Email)
	userName := strings.TrimSpace(request.UserName)
	displayName := strings.TrimSpace(request.DisplayName)

	if email == "" || userName == "" || displayName == "" {
		writeInvalidRequestBody(w, errors.New("email, user name, and display name are required"))
		return
	}

	user, err := service.InTx(
		r.Context(),
		h.database,
		func(s *service.Services) (any, error) {
			user, err := s.Users.Create(r.Context(), email, userName, displayName)
			if err != nil {
				return nil, err
			}
			_, err = s.UserIdentities.Create(
				r.Context(),
				user.ID,
				claims.Subject,
				claims.Issuer,
			)
			if err != nil {
				return nil, err
			}
			return user, nil
		},
	)
	if err != nil {
		var alreadyExists service.AlreadyExistsError
		if errors.As(err, &alreadyExists) {
			http.Error(w, alreadyExists.Error(), http.StatusConflict)
			return
		} else {
			writeInternalServerError(w, err)
			return
		}
	}

	u, ok := user.(*service.User)
	if !ok {
		panic(errors.New("unexpected type returned from transaction"))
	}

	response := CreateUserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt,
	}
	if err := writeJSONResponse(w, http.StatusCreated, response); err != nil {
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
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt,
	}
	if err := writeJSONResponse(w, http.StatusOK, response); err != nil {
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
	if err := writeJSONResponse(w, http.StatusOK, response); err != nil {
		writeInternalServerError(w, err)
	}
}
