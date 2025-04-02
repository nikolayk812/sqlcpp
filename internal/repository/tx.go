package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp/internal/db"
)

func withTx[T any](ctx context.Context, pool *pgxpool.Pool, q *db.Queries, fn func(q *db.Queries) (T, error)) (_ T, txErr error) {
	var zero T

	// If we're already in a transaction (pool is nil), just use the existing queries
	if pool == nil {
		return fn(q)
	}

	// Otherwise, create a new transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, err
	}

	// Ensure proper rollback handling
	defer func() {
		if txErr != nil {
			rollbackErr := tx.Rollback(ctx)
			if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				txErr = errors.Join(txErr, fmt.Errorf("tx.Rollback: %w", rollbackErr))
			}
		}
	}()

	// Create queries with transaction
	qtx := db.New(tx)

	// Execute the function with transaction queries
	result, err := fn(qtx)
	if err != nil {
		return zero, err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}

	return result, nil
}
