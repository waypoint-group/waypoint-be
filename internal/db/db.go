// Package db provides PostgreSQL persistence for the application.
package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database owns the PostgreSQL connection pool and exposes generated queries.
type Database struct {
	pool *pgxpool.Pool
	*sqlc.Queries
}

// New opens a PostgreSQL connection pool and verifies the connection.
func New(ctx context.Context, databaseURL string) (*Database, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return &Database{
		pool:    pool,
		Queries: sqlc.New(pool),
	}, nil
}

func (db *Database) InTx(
	ctx context.Context,
	fn func(*sqlc.Queries) (any, error),
) (result any, inTxErr error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			inTxErr = errors.Join(
				inTxErr,
				fmt.Errorf("rollback transaction: %w", err),
			)
		}
	}()

	qtx := db.WithTx(tx)
	res, err := fn(qtx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return res, nil
}

// Ping verifies that PostgreSQL is reachable.
func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close releases all PostgreSQL connections.
func (db *Database) Close() {
	db.pool.Close()
}
