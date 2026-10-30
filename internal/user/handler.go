package user

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/authn"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) CreateUser(ctx context.Context, req *api.CreateUserRequest) (api.CreateUserRes, error) {
	claims, ok := authn.IdentityClaims(ctx)
	if !ok {
		return nil, errors.New("missing identity in context")
	}
	result, err := h.service.CreateUser(ctx, CreateUserInput{Email: claims.Email, UserName: req.UserName, DisplayName: req.DisplayName, Identity: ExternalIdentity{Issuer: claims.Issuer, Subject: claims.Subject}})
	if err != nil {
		return nil, responseError(err)
	}
	return userResponse(result), nil
}

func (h *Handler) GetUser(ctx context.Context, params api.GetUserParams) (api.GetUserRes, error) {
	result, err := h.service.ReadUser(ctx, uuid.UUID(params.ID))
	if err != nil {
		return nil, responseError(err)
	}
	return userResponse(result), nil
}

func (h *Handler) ListUsers(ctx context.Context) (api.ListUsersRes, error) {
	users, err := h.service.ListUsers(ctx)
	if err != nil {
		return nil, responseError(err)
	}
	result := &api.ListUsersResponse{Users: make([]api.UserResponse, 0, len(users))}
	for i := range users {
		result.Users = append(result.Users, *userResponse(&users[i]))
	}
	return result, nil
}

func (h *Handler) Me(ctx context.Context) (api.MeRes, error) {
	claims, ok := authn.IdentityClaims(ctx)
	if !ok {
		return nil, errors.New("missing identity in context")
	}
	profile, err := h.service.ReadUserProfile(ctx, claims.Issuer, claims.Subject)
	if err != nil {
		return nil, responseError(err)
	}
	result := &api.MeResponse{User: *userResponse(&profile.User), WorkspaceMemberships: make([]api.WorkspaceMembership, 0, len(profile.WorkspaceMemberships))}
	for _, m := range profile.WorkspaceMemberships {
		result.WorkspaceMemberships = append(result.WorkspaceMemberships, api.WorkspaceMembership{WorkspaceID: m.WorkspaceID.String(), WorkspaceName: m.WorkspaceName})
	}
	return result, nil
}

func (h *Handler) DeleteUser(ctx context.Context, params api.DeleteUserParams) (api.DeleteUserRes, error) {
	actor, ok := authn.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	if err := h.service.DeleteUser(ctx, DeleteUserInput{UserID: uuid.UUID(params.ID), ActorID: actor}); err != nil {
		return nil, responseError(err)
	}
	return &api.DeleteUserNoContent{}, nil
}

func userResponse(u *User) *api.UserResponse {
	return &api.UserResponse{ID: u.ID.String(), Email: u.Email, UserName: u.UserName, DisplayName: u.DisplayName, CreatedAt: u.CreatedAt}
}

// responseError maps domain errors to the documented HTTP responses.
func responseError(err error) error {
	status, public := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ErrForbidden):
		status, public = http.StatusForbidden, err.Error()
	case errors.Is(err, ErrNotFound):
		status, public = http.StatusNotFound, err.Error()
	case errors.Is(err, ErrEmailTaken), errors.Is(err, ErrUserNameTaken), errors.Is(err, ErrIdentityTaken):
		status, public = http.StatusConflict, err.Error()
	case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrInvalidUserName), errors.Is(err, ErrInvalidDisplayName), errors.Is(err, ErrInvalidIdentity):
		status, public = http.StatusBadRequest, "invalid request body: "+err.Error()
	}
	if status == http.StatusInternalServerError {
		return err
	}
	return &api.HTTPError{Status: status, Message: public}
}
