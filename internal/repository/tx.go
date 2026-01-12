package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikolayk812/sqlcpp/internal/db"
)

// withTx executes fn within a transaction if the repository was created with a pool,
// or uses the existing transaction if the repository was created with a transaction
func withTx[T any](ctx context.Context, dbtx db.DBTX, fn func(q *db.Queries) (T, error)) (_ T, txErr error) {
	var zero T

	// Check if we're already in a transaction by trying to cast to pgx.Tx
	if tx, ok := dbtx.(pgx.Tx); ok {
		// Already in a transaction, just use it
		q := db.New(tx)
		return fn(q)
	}

	// Must be a pool, create a new transaction
	pool, ok := dbtx.(*pgxpool.Pool)
	if !ok {
		return zero, fmt.Errorf("dbtx is neither pgx.Tx nor *pgxpool.Pool: %T", dbtx)
	}

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
