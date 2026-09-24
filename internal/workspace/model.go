package workspace

import (
	"time"
	"uuid"
)

// CreateWorkspaceInput identifies the creator and the requested workspace name.
type CreateWorkspaceInput struct {
	Name    string
	ActorID uuid.UUID
}

// ReadWorkspaceInput identifies a workspace and the member requesting its details.
type ReadWorkspaceInput struct {
	WorkspaceID uuid.UUID
	ActorID     uuid.UUID
}

// AddWorkspaceMemberInput identifies the acting user and the user to add.
type AddWorkspaceMemberInput struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	ActorID     uuid.UUID
}

// RemoveWorkspaceMemberInput identifies the acting user and the user to remove.
type RemoveWorkspaceMemberInput struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	ActorID     uuid.UUID
}

// DeleteWorkspaceInput identifies the workspace and the user requesting deletion.
type DeleteWorkspaceInput struct {
	WorkspaceID uuid.UUID
	ActorID     uuid.UUID
}

// CreateWorkspaceRequest contains the fields accepted when creating a workspace.
type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

// Workspace contains workspace details and its current membership.
type Workspace struct {
	ID        uuid.UUID                `json:"id"`
	Name      string                   `json:"name"`
	Members   []WorkspaceMemberSummary `json:"members"`
	CreatedAt time.Time                `json:"created_at"`
}

// WorkspaceMemberSummary describes a member within its containing workspace.
type WorkspaceMemberSummary struct {
	UserID uuid.UUID     `json:"user_id"`
	Role   WorkspaceRole `json:"role"`
}

// WorkspaceMember associates a user with a workspace and a role.
type WorkspaceMember struct {
	WorkspaceID uuid.UUID     `json:"workspace_id"`
	UserID      uuid.UUID     `json:"user_id"`
	Role        WorkspaceRole `json:"role"`
}

// WorkspaceRole determines a member’s permissions within a workspace.
// NOTE: The admin role should be ignored for now; only owners can manage a workspace.
type WorkspaceRole string

const (
	// WorkspaceRoleOwner may add and remove members and delete the workspace.
	WorkspaceRoleOwner WorkspaceRole = "owner"
	// WorkspaceRoleMember may view the workspace.
	WorkspaceRoleMember WorkspaceRole = "member"
)
