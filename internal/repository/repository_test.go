package repository_test

import (
	"context"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startPostgres(ctx context.Context) (testcontainers.Container, string, error) {
	postgresContainer, err := postgres.Run(ctx, "postgres:17.4-alpine",
		postgres.BasicWaitStrategies(),
		postgres.WithInitScripts(
			"../migrations/01_cart_items.up.sql",
			"../migrations/02_orders.up.sql"),
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
