// Package workspace manages workspaces and their membership permissions.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Service manages workspaces and workspace membership.
//
// Every workspace has exactly one owner and therefore at least one member.
// This invariant is preserved by the following rules:
//   - Workspace creation adds the owner atomically
//   - Workspace owners cannot be removed from the workspace
//   - Deleting the owner's account deletes their workspaces atomically
type Service struct {
	database *db.Database
}

// NewService constructs a workspace service.
func NewService(database *db.Database) *Service {
	return &Service{database: database}
}

// CreateWorkspace atomically creates a workspace and its owner membership.
func (s *Service) CreateWorkspace(
	ctx context.Context,
	input CreateWorkspaceInput,
) (_ *Workspace, createWorkspaceError error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			createWorkspaceError = errors.Join(createWorkspaceError, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	wsRow, err := tx.CreateWorkspace(
		ctx,
		sqlc.CreateWorkspaceParams{ID: uuid.NewV7(), Name: name},
	)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	owner, err := tx.AddWorkspaceMember(
		ctx,
		sqlc.AddWorkspaceMemberParams{
			WorkspaceID: wsRow.ID,
			UserID:      input.ActorID,
			Role:        string(WorkspaceRoleOwner),
		},
	)
	if err != nil {
		if db.IsForeignKeyViolation(err, "workspace_members_user_id_fkey") {
			return nil, fmt.Errorf("add workspace owner: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return workspaceFromRows(wsRow, []sqlc.WorkspaceMember{owner}), nil
}

// ReadWorkspace returns workspace details to a current member.
func (s *Service) ReadWorkspace(
	ctx context.Context,
	input ReadWorkspaceInput,
) (_ *Workspace, readWorkspaceError error) {
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workspace transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			readWorkspaceError = errors.Join(readWorkspaceError, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Keep the permission check and returned membership list consistent with
	// concurrent additions, removals, and workspace deletion.
	wsRow, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock workspace: %w", err)
	}

	members, err := tx.ListWorkspaceMembers(ctx, input.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}

	isMember := false
	for _, m := range members {
		if m.UserID == input.ActorID {
			isMember = true
			break
		}
	}
	if !isMember {
		return nil, ErrForbidden
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return workspaceFromRows(wsRow, members), nil
}

// AddWorkspaceMember allows the owner to add a registered user as a member.
func (s *Service) AddWorkspaceMember(
	ctx context.Context,
	input AddWorkspaceMemberInput,
) (_ *WorkspaceMember, addWorkspaceMemberError error) {
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			addWorkspaceMemberError = errors.Join(addWorkspaceMemberError, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	wsRow, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock workspace: %w", err)
	}

	actor, err := tx.SelectWorkspaceMember(
		ctx,
		sqlc.SelectWorkspaceMemberParams{WorkspaceID: wsRow.ID, UserID: input.ActorID},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrForbidden
		}

		return nil, fmt.Errorf("read actor membership: %w", err)
	}
	if actor.Role != string(WorkspaceRoleOwner) {
		return nil, ErrForbidden
	}

	member, err := tx.AddWorkspaceMember(
		ctx,
		sqlc.AddWorkspaceMemberParams{
			WorkspaceID: wsRow.ID,
			UserID:      input.UserID,
			Role:        string(WorkspaceRoleMember),
		},
	)
	if err != nil {
		if db.IsUniqueViolation(err, "workspace_members_pkey") {
			return nil, ErrAlreadyMember
		}
		if db.IsForeignKeyViolation(err, "workspace_members_user_id_fkey") {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("add workspace member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	result := memberFromRow(member)
	return &result, nil
}

// RemoveWorkspaceMember allows the owner to remove another user and their channel memberships.
func (s *Service) RemoveWorkspaceMember(
	ctx context.Context,
	input RemoveWorkspaceMemberInput,
) (removeWorkspaceMemberError error) {
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			removeWorkspaceMemberError = errors.Join(
				fmt.Errorf("rollback transaction: %w", err),
				removeWorkspaceMemberError,
			)
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	wsRow, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWorkspaceNotFound
		}

		return fmt.Errorf("lock workspace: %w", err)
	}
	// Make sure action was performed by the owner.
	actor, err := tx.SelectWorkspaceMember(
		ctx,
		sqlc.SelectWorkspaceMemberParams{
			WorkspaceID: wsRow.ID, UserID: input.ActorID,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}

		return fmt.Errorf("read actor membership: %w", err)
	}
	if actor.Role != string(WorkspaceRoleOwner) {
		return ErrForbidden
	}
	// Prevent the owner from removing themselves, which would invalidate
	// the "a workspace always has at least one member (the owner) contract".
	if input.UserID == input.ActorID {
		return ErrOwnerRemoval
	}
	// Remove the workspace member.
	removed, err := tx.RemoveWorkspaceMember(
		ctx,
		sqlc.RemoveWorkspaceMemberParams{
			WorkspaceID: wsRow.ID, UserID: input.UserID,
		},
	)
	if err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}
	if removed == 0 {
		return ErrWorkspaceMemberNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DeleteWorkspace allows the owner to delete a workspace, even when other members remain.
//
// Memberships, channels, and channel messages are deleted by cascading foreign keys.
func (s *Service) DeleteWorkspace(
	ctx context.Context,
	input DeleteWorkspaceInput,
) (deleteWorkspaceError error) {
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			deleteWorkspaceError = errors.Join(deleteWorkspaceError, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	wsRow, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWorkspaceNotFound
		}

		return fmt.Errorf("lock workspace: %w", err)
	}
	// Make sure action was performed by the owner.
	actor, err := tx.SelectWorkspaceMember(ctx, sqlc.SelectWorkspaceMemberParams{WorkspaceID: wsRow.ID, UserID: input.ActorID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}

		return fmt.Errorf("read actor membership: %w", err)
	}
	if actor.Role != string(WorkspaceRoleOwner) {
		return ErrForbidden
	}
	// Delete the workspace.
	removed, err := tx.DeleteWorkspace(ctx, wsRow.ID)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if removed == 0 {
		return ErrWorkspaceNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// memberFromRow converts a database membership into its public representation.
func memberFromRow(row sqlc.WorkspaceMember) WorkspaceMember {
	return WorkspaceMember{
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        WorkspaceRole(row.Role),
	}
}

// workspaceFromRows combines workspace details and an ordered membership list.
func workspaceFromRows(row sqlc.Workspace, members []sqlc.WorkspaceMember) *Workspace {
	result := &Workspace{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		Members:   make([]WorkspaceMemberSummary, 0, len(members)),
	}
	for _, member := range members {
		result.Members = append(result.Members, WorkspaceMemberSummary{
			UserID: member.UserID,
			Role:   WorkspaceRole(member.Role),
		})
	}
	return result
}
