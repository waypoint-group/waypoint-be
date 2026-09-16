package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"uuid"

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
	if err := decodeJSONRequest(w, r, &request); err != nil {
		invalidRequestBody(w, err)
		return
	}

	u, err := h.service.CreateUser(r.Context(), CreateUserInput{
		Email:       request.Email,
		UserName:    request.UserName,
		DisplayName: request.DisplayName,
		Identity:    ExternalIdentity{Issuer: claims.Issuer, Subject: claims.Subject},
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken):
			http.Error(w, ErrEmailTaken.Error(), http.StatusConflict)
		case errors.Is(err, ErrUserNameTaken):
			http.Error(w, ErrUserNameTaken.Error(), http.StatusConflict)
		case errors.Is(err, ErrIdentityTaken):
			http.Error(w, ErrIdentityTaken.Error(), http.StatusConflict)
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidUserName),
			errors.Is(err, ErrInvalidDisplayName), errors.Is(err, ErrInvalidIdentity):
			invalidRequestBody(w, err)
		default:
			internalServerError(w, err)
		}
		return
	}

	response := CreateUserResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		UserName:    u.UserName,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt,
	}
	if err := writeJSONResponse(w, http.StatusCreated, response); err != nil {
		internalServerError(w, err)
	}
}

// GetUser retrieves the user identified by the request path.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ID, err := uuid.Parse(r.PathValue("id"))
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

		internalServerError(w, err)
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
		internalServerError(w, err)
	}
}

// ListUsers writes all users returned by the accounts service.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		internalServerError(w, err)
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
		internalServerError(w, err)
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
			internalServerError(w, err)
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
		internalServerError(w, err)
	}
}

const maxRequestBodyBytes = 1 << 20

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(payload)
	return err
}

func internalServerError(w http.ResponseWriter, err error) {
	log.Printf("internal server error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func invalidRequestBody(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
}
