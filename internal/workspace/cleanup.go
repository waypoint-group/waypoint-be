package workspace

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// runWorkspaceCleanup selects on the service context, which Close cancels.
func (s *Service) runWorkspaceCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := s.deleteEmptyWorkspaces(ctx)
			if err != nil && ctx.Err() == nil {
				log.Printf("workspace cleanup: %v", err)
			}
		}
	}
}

// deleteEmptyWorkspaces deletes up to 100 empty workspaces and their channels.
// Busy workspaces are skipped until a later run. Membership additions acquire
// the same workspace locks, so the membership recheck is safe at READ COMMITTED.
func (s *Service) deleteEmptyWorkspaces(ctx context.Context) (deleted int64, resultErr error) {
	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin workspace cleanup: %w", err)
	}
	defer func() {
		if err := tx.RollbackUnlessClosed(ctx); err != nil {
			resultErr = errors.Join(
				fmt.Errorf("rollback transaction: %w", err),
				err,
			)
		}
	}()
	ids, err := tx.LockEmptyWorkspaces(ctx)
	if err != nil {
		return 0, fmt.Errorf("lock empty workspaces: %w", err)
	}
	// Use a new statement after locking so additions committed since the initial
	// candidate snapshot are visible before deciding whether to delete.
	deleted, err = tx.DeleteEmptyWorkspaces(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("delete empty workspaces: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit workspace cleanup: %w", err)
	}
	return deleted, nil
}
