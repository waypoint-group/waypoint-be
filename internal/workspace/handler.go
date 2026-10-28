package workspace

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	googleuuid "github.com/google/uuid"
	"github.com/waypoint-group/waypoint-be/internal/api"
	"github.com/waypoint-group/waypoint-be/internal/user"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) CreateWorkspace(ctx context.Context, req *api.CreateWorkspaceRequest) (api.CreateWorkspaceRes, error) {
	actor, ok := user.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	result, err := h.service.CreateWorkspace(ctx, CreateWorkspaceInput{Name: req.Name, ActorID: actor})
	if err != nil {
		return nil, responseError(err)
	}
	response := workspaceResponse(result)
	return &response, nil
}

func (h *Handler) GetWorkspace(ctx context.Context, params api.GetWorkspaceParams) (api.GetWorkspaceRes, error) {
	actor, ok := user.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	result, err := h.service.ReadWorkspace(ctx, ReadWorkspaceInput{WorkspaceID: uuid.UUID(params.WorkspaceID), ActorID: actor})
	if err != nil {
		return nil, responseError(err)
	}
	response := workspaceResponse(result)
	return &response, nil
}

func (h *Handler) AddWorkspaceMember(ctx context.Context, params api.AddWorkspaceMemberParams) (api.AddWorkspaceMemberRes, error) {
	actor, ok := user.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	result, err := h.service.AddWorkspaceMember(ctx, AddWorkspaceMemberInput{WorkspaceID: uuid.UUID(params.WorkspaceID), UserID: uuid.UUID(params.UserID), ActorID: actor})
	if err != nil {
		return nil, responseError(err)
	}
	return &api.WorkspaceMember{WorkspaceID: googleuuid.UUID(result.WorkspaceID), UserID: googleuuid.UUID(result.UserID), Role: api.WorkspaceMemberRole(result.Role)}, nil
}

func (h *Handler) RemoveWorkspaceMember(ctx context.Context, params api.RemoveWorkspaceMemberParams) (api.RemoveWorkspaceMemberRes, error) {
	actor, ok := user.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	if err := h.service.RemoveWorkspaceMember(ctx, RemoveWorkspaceMemberInput{WorkspaceID: uuid.UUID(params.WorkspaceID), UserID: uuid.UUID(params.UserID), ActorID: actor}); err != nil {
		return nil, responseError(err)
	}
	return &api.RemoveWorkspaceMemberNoContent{}, nil
}

func (h *Handler) DeleteWorkspace(ctx context.Context, params api.DeleteWorkspaceParams) (api.DeleteWorkspaceRes, error) {
	actor, ok := user.ActorID(ctx)
	if !ok {
		return nil, errors.New("missing actor in context")
	}
	if err := h.service.DeleteWorkspace(ctx, DeleteWorkspaceInput{WorkspaceID: uuid.UUID(params.WorkspaceID), ActorID: actor}); err != nil {
		return nil, responseError(err)
	}
	return &api.DeleteWorkspaceNoContent{}, nil
}

// responseError maps domain errors to the documented HTTP responses.
func responseError(err error) error {
	status, public := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ErrForbidden):
		status, public = http.StatusForbidden, ErrForbidden.Error()
	case errors.Is(err, ErrInvalidID):
		status, public = http.StatusBadRequest, ErrInvalidID.Error()
	case errors.Is(err, ErrInvalidName):
		status, public = http.StatusBadRequest, ErrInvalidName.Error()
	case errors.Is(err, ErrWorkspaceNotFound):
		status, public = http.StatusNotFound, ErrWorkspaceNotFound.Error()
	case errors.Is(err, ErrWorkspaceMemberNotFound):
		status, public = http.StatusNotFound, ErrWorkspaceMemberNotFound.Error()
	case errors.Is(err, ErrUserNotFound):
		status, public = http.StatusNotFound, ErrUserNotFound.Error()
	case errors.Is(err, ErrAlreadyMember):
		status, public = http.StatusConflict, ErrAlreadyMember.Error()
	case errors.Is(err, ErrOwnerRemoval):
		status, public = http.StatusConflict, ErrOwnerRemoval.Error()
	}
	if status == http.StatusInternalServerError {
		return err
	}
	return &api.HTTPError{Status: status, Message: public}
}

// workspaceResponse maps domain values to the generated wire contract.
func workspaceResponse(workspace *Workspace) api.Workspace {
	members := make([]api.WorkspaceMemberSummary, 0, len(workspace.Members))
	for _, member := range workspace.Members {
		members = append(members, api.WorkspaceMemberSummary{
			UserID: googleuuid.UUID(member.UserID),
			Role:   api.WorkspaceMemberSummaryRole(member.Role),
		})
	}
	return api.Workspace{
		ID: googleuuid.UUID(workspace.ID), Name: workspace.Name,
		CreatedAt: workspace.CreatedAt, Members: members,
	}
}
