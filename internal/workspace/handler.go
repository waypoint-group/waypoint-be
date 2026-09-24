package workspace

import (
	"errors"
	"log"
	"net/http"
	"uuid"

	"github.com/waypoint-group/waypoint-be/internal/httpx"
	"github.com/waypoint-group/waypoint-be/internal/middleware"
	"github.com/waypoint-group/waypoint-be/internal/user"
)

// Handler serves authenticated workspace and membership operations over HTTP.
type Handler struct {
	service   *Service
	users     *user.Service
	jwtConfig middleware.JWTConfig
}

// NewHandler constructs a handler that resolves verified identities to local users.
func NewHandler(service *Service, users *user.Service, jwtConfig middleware.JWTConfig) *Handler {
	return &Handler{service: service, users: users, jwtConfig: jwtConfig}
}

// CreateWorkspace creates a workspace owned by the authenticated user.
func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}

	var request CreateWorkspaceRequest
	if err := httpx.ReadJSON(r, &request); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.service.CreateWorkspace(r.Context(), CreateWorkspaceInput{Name: request.Name, ActorID: actorID})
	if err != nil {
		writeError(w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, result); err != nil {
		writeError(w, err)
		return
	}
}

// GetWorkspace returns workspace details to an authenticated member.
func (h *Handler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}

	workspaceID, err := httpx.PathID(r, "workspace_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.service.ReadWorkspace(r.Context(), ReadWorkspaceInput{WorkspaceID: workspaceID, ActorID: actorID})
	if err != nil {
		writeError(w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, result); err != nil {
		writeError(w, err)
		return
	}
}

// AddWorkspaceMember adds the user in the path with the member role.
func (h *Handler) AddWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}

	workspaceID, err := httpx.PathID(r, "workspace_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}
	userID, err := httpx.PathID(r, "user_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.service.AddWorkspaceMember(r.Context(), AddWorkspaceMemberInput{WorkspaceID: workspaceID, UserID: userID, ActorID: actorID})
	if err != nil {
		writeError(w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, result); err != nil {
		writeError(w, err)
		return
	}
}

// RemoveWorkspaceMember removes the user in the path, subject to owner permissions.
func (h *Handler) RemoveWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}

	workspaceID, err := httpx.PathID(r, "workspace_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}
	userID, err := httpx.PathID(r, "user_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.RemoveWorkspaceMember(r.Context(), RemoveWorkspaceMemberInput{WorkspaceID: workspaceID, UserID: userID, ActorID: actorID}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteWorkspace deletes a workspace when the authenticated owner is its only member.
func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.actorID(w, r)
	if !ok {
		return
	}

	workspaceID, err := httpx.PathID(r, "workspace_id")
	if err != nil {
		http.Error(w, ErrInvalidID.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteWorkspace(r.Context(), DeleteWorkspaceInput{WorkspaceID: workspaceID, ActorID: actorID}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// actorID derives authority from the verified token.
func (h *Handler) actorID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	_, claims, err := middleware.ExtractAndValidateJWT(r, h.jwtConfig)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return uuid.UUID{}, false
	}

	actor, err := h.users.ReadUserByIdentity(r.Context(), claims.Issuer, claims.Subject)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			http.Error(w, ErrForbidden.Error(), http.StatusForbidden)
		} else {
			log.Printf("resolve workspace actor: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return uuid.UUID{}, false
	}

	return actor.ID, true
}

// writeError maps domain sentinels to HTTP statuses and hides unexpected causes.
func writeError(w http.ResponseWriter, err error) {
	status, public := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ErrInvalidID):
		status, public = http.StatusBadRequest, ErrInvalidID.Error()
	case errors.Is(err, ErrInvalidName):
		status, public = http.StatusBadRequest, ErrInvalidName.Error()
	case errors.Is(err, ErrForbidden):
		status, public = http.StatusForbidden, ErrForbidden.Error()
	case errors.Is(err, ErrNotFound):
		status, public = http.StatusNotFound, ErrNotFound.Error()
	case errors.Is(err, ErrMemberNotFound):
		status, public = http.StatusNotFound, ErrMemberNotFound.Error()
	case errors.Is(err, ErrUserNotFound):
		status, public = http.StatusNotFound, ErrUserNotFound.Error()
	case errors.Is(err, ErrAlreadyMember):
		status, public = http.StatusConflict, ErrAlreadyMember.Error()
	case errors.Is(err, ErrOwnerRemoval):
		status, public = http.StatusConflict, ErrOwnerRemoval.Error()
	case errors.Is(err, ErrWorkspaceNotEmpty):
		status, public = http.StatusConflict, ErrWorkspaceNotEmpty.Error()
	}
	if status == http.StatusInternalServerError {
		log.Printf("workspace error: %v", err)
	}
	http.Error(w, public, status)
}
