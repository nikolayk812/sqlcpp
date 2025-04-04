-- name: GetOrder :one
SELECT id, owner_id, created_at, updated_at, url, status, tags
FROM orders
WHERE id = $1;

-- name: InsertOrder :one
INSERT INTO orders (owner_id, url, tags)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetOrderItems :many
SELECT product_id, price_amount, price_currency, created_at
FROM order_items
WHERE order_id = $1;

-- name: InsertOrderItem :exec
INSERT INTO order_items (order_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4);

-- name: GetOrderJoinItems :many
SELECT
    o.id, o.owner_id, o.created_at, o.updated_at, o.url, o.status, o.tags,
    oi.product_id, oi.price_amount, oi.price_currency
FROM orders o
         JOIN order_items oi ON o.id = oi.order_id
WHERE o.id = $1;