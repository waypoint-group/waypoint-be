// Package workspace manages workspaces and their membership permissions.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Service manages workspaces and workspace membership.
type Service struct {
	database      *db.Database
	cleanupCancel context.CancelFunc
	cleanupDone   chan struct{}
}

// NewService constructs a workspace service and starts periodic workspace cleanup.
func NewService(database *db.Database, cleanupInterval time.Duration) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	service := &Service{
		database:      database,
		cleanupCancel: cancel,
		cleanupDone:   make(chan struct{}),
	}
	go func() {
		defer close(service.cleanupDone)
		service.runWorkspaceCleanup(ctx, cleanupInterval)
	}()
	return service
}

// Close cancels cleanup and waits for in-flight work to release its connection.
// It is safe to call repeatedly; the database remains caller-owned.
func (s *Service) Close() {
	s.cleanupCancel()
	<-s.cleanupDone
}

// CreateWorkspace atomically creates a workspace and its owner membership.
func (s *Service) CreateWorkspace(
	ctx context.Context,
	input CreateWorkspaceInput,
) (_ *Workspace, resultErr error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidName
	}
	if input.ActorID == (uuid.UUID{}) {
		return nil, ErrInvalidID
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	row, err := tx.CreateWorkspace(
		ctx,
		sqlc.CreateWorkspaceParams{ID: uuid.NewV7(), Name: name},
	)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	owner, err := tx.AddWorkspaceMember(
		ctx,
		sqlc.AddWorkspaceMemberParams{
			WorkspaceID: row.ID,
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

	return workspaceFromRows(row, []sqlc.WorkspaceMember{owner}), nil
}

// ReadWorkspace returns workspace details to a current member.
func (s *Service) ReadWorkspace(
	ctx context.Context,
	input ReadWorkspaceInput,
) (_ *Workspace, resultErr error) {
	if input.WorkspaceID == (uuid.UUID{}) || input.ActorID == (uuid.UUID{}) {
		return nil, ErrInvalidID
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workspace transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Keep the permission check and returned membership list consistent with
	// concurrent additions, removals, and workspace deletion.
	row, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
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

	return workspaceFromRows(row, members), nil
}

// AddWorkspaceMember allows the owner to add a registered user as a member.
func (s *Service) AddWorkspaceMember(
	ctx context.Context,
	input AddWorkspaceMemberInput,
) (_ *WorkspaceMember, resultErr error) {
	if input.WorkspaceID == (uuid.UUID{}) ||
		input.ActorID == (uuid.UUID{}) ||
		input.UserID == (uuid.UUID{}) {
		return nil, ErrInvalidID
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	row, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock workspace: %w", err)
	}

	actor, err := tx.SelectWorkspaceMember(
		ctx,
		sqlc.SelectWorkspaceMemberParams{WorkspaceID: row.ID, UserID: input.ActorID},
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
			WorkspaceID: row.ID,
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
) (resultErr error) {
	if input.WorkspaceID == (uuid.UUID{}) ||
		input.ActorID == (uuid.UUID{}) ||
		input.UserID == (uuid.UUID{}) {
		return ErrInvalidID
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// Rollback must still release locks if the request was cancelled.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	row, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}

		return fmt.Errorf("lock workspace: %w", err)
	}

	actor, err := tx.SelectWorkspaceMember(
		ctx,
		sqlc.SelectWorkspaceMemberParams{
			WorkspaceID: row.ID, UserID: input.ActorID,
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
	if input.UserID == input.ActorID {
		return ErrOwnerRemoval
	}

	_, err = tx.SelectWorkspaceMember(
		ctx,
		sqlc.SelectWorkspaceMemberParams{
			WorkspaceID: row.ID, UserID: input.UserID,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemberNotFound
		}

		return fmt.Errorf("read target membership: %w", err)
	}

	err = tx.RemoveWorkspaceMember(
		ctx,
		sqlc.RemoveWorkspaceMemberParams{
			WorkspaceID: row.ID, UserID: input.UserID,
		},
	)
	if err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DeleteWorkspace allows the owner to delete a workspace once no other members remain.
// Channels and messages are deleted by the database's cascading foreign keys.
func (s *Service) DeleteWorkspace(ctx context.Context, input DeleteWorkspaceInput) (resultErr error) {
	if input.WorkspaceID == (uuid.UUID{}) || input.ActorID == (uuid.UUID{}) {
		return ErrInvalidID
	}

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("rollback transaction: %w", err))
		}
	}()

	// Lock before checking membership so concurrent workspace operations cannot
	// invalidate the permission or membership checks before commit.
	row, err := tx.LockWorkspaceRow(ctx, input.WorkspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}

		return fmt.Errorf("lock workspace: %w", err)
	}

	actor, err := tx.SelectWorkspaceMember(ctx, sqlc.SelectWorkspaceMemberParams{WorkspaceID: row.ID, UserID: input.ActorID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}

		return fmt.Errorf("read actor membership: %w", err)
	}
	if actor.Role != string(WorkspaceRoleOwner) {
		return ErrForbidden
	}

	members, err := tx.ListWorkspaceMembers(ctx, row.ID)
	if err != nil {
		return fmt.Errorf("list workspace members: %w", err)
	}
	if len(members) != 1 {
		return ErrWorkspaceNotEmpty
	}

	if err := tx.DeleteWorkspace(ctx, row.ID); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
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
