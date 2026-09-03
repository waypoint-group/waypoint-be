// Package db provides PostgreSQL persistence for the application.
package db

import (
	"context"
	"fmt"

	"github.com/waypoint-group/waypoint-be/internal/db/sqlc"

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

// Ping verifies that PostgreSQL is reachable.
func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close releases all PostgreSQL connections.
func (db *Database) Close() {
	db.pool.Close()
}
