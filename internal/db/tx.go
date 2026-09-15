package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"
)

// Transaction exposes queries bound to a PostgreSQL transaction.
type Transaction struct {
	*sqlc.Queries
	tx pgx.Tx
}

// Rollback aborts the transaction, returning pgx.ErrTxClosed if already closed.
func (tx *Transaction) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}

// RollbackUnlessClosed aborts an open transaction and ignores pgx.ErrTxClosed.
func (tx *Transaction) RollbackUnlessClosed(ctx context.Context) error {
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return err
	}
	return nil
}

// Commit commits the transaction and returns any database error.
func (tx *Transaction) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}
