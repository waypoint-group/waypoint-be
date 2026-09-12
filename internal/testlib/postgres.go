//go:build integration

package testlib

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var postgresImage = "postgres:18"

type Postgres struct {
	ctr *postgres.PostgresContainer
	URL string
}

func NewPostgres() (*Postgres, error) {
	parent := context.Background()
	ctr, err := postgres.Run(
		parent,
		postgresImage,
		postgres.WithDatabase("waypoint-test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secret"),
		postgres.BasicWaitStrategies(),
	)
	pg := &Postgres{ctr: ctr}
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	pg.URL, err = ctr.ConnectionString(
		parent,
		// Disable SSL for tests.
		"sslmode=disable",
	)
	if err != nil {
		pg.Close()
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return pg, nil
}

func (p *Postgres) Close() {
	if p.ctr != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := p.ctr.Terminate(cleanupCtx)
		if err != nil {
			fmt.Printf("failed to terminate container: %v\n", err)
		}
	}
}
