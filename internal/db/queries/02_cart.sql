-- name: GetOrder :one
SELECT id, owner_id, created_at, updated_at
FROM orders
WHERE id = $1;

-- name: InsertOrder :one
INSERT INTO orders (owner_id)
VALUES ($1)
RETURNING id;

-- name: GetOrderItems :many
SELECT product_id, price_amount, price_currency, created_at
FROM order_items
WHERE order_id = $1;

-- name: InsertOrderItem :exec
INSERT INTO order_items (order_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4);