// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: 01_cart.sql

package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const addItem = `-- name: AddItem :exec
INSERT INTO cart_items (owner_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4)
ON CONFLICT (owner_id, product_id) DO UPDATE
    SET price_amount = EXCLUDED.price_amount, price_currency = EXCLUDED.price_currency
`

type AddItemParams struct {
	OwnerID       string
	ProductID     uuid.UUID
	PriceAmount   decimal.Decimal
	PriceCurrency string
}

func (q *Queries) AddItem(ctx context.Context, arg AddItemParams) error {
	_, err := q.db.Exec(ctx, addItem,
		arg.OwnerID,
		arg.ProductID,
		arg.PriceAmount,
		arg.PriceCurrency,
	)
	return err
}

const deleteItem = `-- name: DeleteItem :execrows
DELETE FROM cart_items WHERE owner_id = $1 AND product_id = $2
`

type DeleteItemParams struct {
	OwnerID   string
	ProductID uuid.UUID
}

func (q *Queries) DeleteItem(ctx context.Context, arg DeleteItemParams) (int64, error) {
	result, err := q.db.Exec(ctx, deleteItem, arg.OwnerID, arg.ProductID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const getCart = `-- name: GetCart :many
SELECT product_id, price_amount, price_currency, created_at
FROM cart_items
WHERE owner_id = $1
`

type GetCartRow struct {
	ProductID     uuid.UUID
	PriceAmount   decimal.Decimal
	PriceCurrency string
	CreatedAt     time.Time
}

func (q *Queries) GetCart(ctx context.Context, ownerID string) ([]GetCartRow, error) {
	rows, err := q.db.Query(ctx, getCart, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCartRow
	for rows.Next() {
		var i GetCartRow
		if err := rows.Scan(
			&i.ProductID,
			&i.PriceAmount,
			&i.PriceCurrency,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
