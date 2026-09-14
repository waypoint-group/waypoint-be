//go:build integration

package testlib

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var postgresImage = "postgres:18"

type Postgres struct {
	ctr *postgres.PostgresContainer
	URL string
}

func NewPostgres() (*Postgres, error) {
	parent, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ctr, err := postgres.Run(
		parent,
		postgresImage,
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secret"),
		postgres.BasicWaitStrategies(),
	)

	pg := &Postgres{ctr: ctr}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to start container: %w", err), pg.Close())
	}

	pg.URL, err = ctr.ConnectionString(
		parent,
		// Disable SSL for tests.
		"sslmode=disable",
	)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to get connection string: %w", err), pg.Close())
	}
	return pg, nil
}

func (p *Postgres) Close() error {
	if p.ctr == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.ctr.Terminate(ctx); err != nil {
		return fmt.Errorf("terminate PostgreSQL container: %w", err)
	}
	return nil
}

func (p *Postgres) CreateDatabase(ctx context.Context) (*TestDatabase, error) {
	conn, err := pgx.Connect(ctx, p.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL admin database: %w", err)
	}

	dbName := "pg_test_" + uuid.New().String()
	dbIdent := pgx.Identifier{dbName}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+dbIdent); err != nil {
		return nil, errors.Join(
			fmt.Errorf("create test database: %w", err),
			closeAdminConnection(conn),
		)
	}

	databaseURL, err := url.Parse(p.URL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL URL: %w", err)
	}
	databaseURL.Path = "/" + dbName
	db := &TestDatabase{
		URL:             databaseURL.String(),
		name:            dbName,
		adminConnection: conn,
	}

	return db, nil
}

// TestDatabase owns one database inside the shared PostgreSQL container.
type TestDatabase struct {
	URL             string
	name            string
	adminConnection *pgx.Conn
}

func (db *TestDatabase) Close(ctx context.Context) (closeErr error) {
	defer func() { closeErr = errors.Join(closeErr, closeAdminConnection(db.adminConnection)) }()
	dbIdent := pgx.Identifier{db.name}.Sanitize()
	if _, err := db.adminConnection.Exec(ctx, "DROP DATABASE IF EXISTS "+dbIdent); err != nil {
		return fmt.Errorf("drop test database %s: %w", db.name, err)
	}
	return nil
}

func closeAdminConnection(conn *pgx.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := conn.Close(ctx); err != nil {
		return fmt.Errorf("close PostgreSQL admin connection: %w", err)
	}
	return nil
}
