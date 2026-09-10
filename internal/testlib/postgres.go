//go:build integration

package testlib

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var postgresImage = "postgres:18"

type Postgres struct {
	*postgres.PostgresContainer
	URL string
}

func NewPostgres() (*Postgres, error) {
	postgresContainer, err := postgres.Run(
		context.Background(),
		postgresImage,
		postgres.WithDatabase("waypoint-test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("secret"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	connectionString, err := postgresContainer.ConnectionString(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &Postgres{
		PostgresContainer: postgresContainer,
		URL:               connectionString,
	}, nil
}
