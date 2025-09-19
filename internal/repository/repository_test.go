package repository_test

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startPostgres(ctx context.Context) (*postgres.PostgresContainer, string, error) {
	postgresContainer, err := postgres.Run(ctx, "postgres:17.6-alpine3.22",
		postgres.BasicWaitStrategies(),
		postgres.WithInitScripts(
			"../migrations/01_orders.up.sql",
			"../migrations/02_orders_complex.up.sql"),
	)
	if err != nil {
		return nil, "", fmt.Errorf("postgres.Run: %w", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", fmt.Errorf("pc.ConnectionString: %w", err)
	}

	return postgresContainer, connStr, nil
}
