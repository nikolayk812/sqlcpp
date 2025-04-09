-- name: GetOrder :one
SELECT id,
       owner_id,
       created_at,
       updated_at,
       url,
       status,
       tags,
       payload,
       payloadb,
       deleted_at,
       price_amount,
       price_currency
FROM orders
WHERE id = $1
  AND deleted_at IS NULL;

-- name: InsertOrder :one
INSERT INTO orders (owner_id, url, tags, payload, payloadb, price_amount, price_currency)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: GetOrderItems :many
SELECT product_id, price_amount, price_currency, created_at
FROM order_items
WHERE order_id = $1
  AND deleted_at IS NULL;

-- name: InsertOrderItem :exec
INSERT INTO order_items (order_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4);

-- name: DeleteOrder :execresult
DELETE
FROM orders
WHERE id = $1;

-- name: DeleteOrderItems :execresult
DELETE
FROM order_items
WHERE order_id = $1;

-- name: SoftDeleteOrder :execresult
UPDATE orders
SET deleted_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: SoftDeleteOrderItem :execresult
UPDATE order_items
SET deleted_at = NOW()
WHERE order_id = $1
  AND product_id = $2
  AND deleted_at IS NULL;

-- name: SetOrderUpdated :execresult
UPDATE orders
SET updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: GetOrderJoinItems :many
SELECT o.id,
       o.owner_id,
       o.created_at,
       o.updated_at,
       o.url,
       o.status,
       o.tags,
       o.payload,
       o.payloadb,
       o.price_amount,
       o.price_currency,
       oi.product_id,
       oi.price_amount   AS item_price_amount,
       oi.price_currency AS item_price_currency
FROM orders o
         JOIN order_items oi ON o.id = oi.order_id
WHERE o.id = $1
  ANd o.deleted_at IS NULL
  AND oi.deleted_at IS NULL;

-- name: SearchOrders :many
SELECT o.id,
       o.owner_id,
       o.created_at,
       o.updated_at,
       o.url,
       o.status,
       o.tags,
       o.payload,
       o.payloadb,
       o.price_amount,
       o.price_currency,
       oi.product_id,
       oi.price_amount   AS item_price_amount,
       oi.price_currency AS item_price_currency
FROM orders o
         JOIN order_items oi ON o.id = oi.order_id
WHERE (
          (@ids::UUID[] IS NULL OR o.id = ANY (@ids))
              AND
          (@owner_ids::VARCHAR[] IS NULL OR o.owner_id = ANY (@owner_ids))
              AND
          (@url_patterns::TEXT[] IS NULL OR EXISTS (SELECT 1
                                                    FROM unnest(@url_patterns) AS url_pattern
                                                    WHERE o.url ILIKE '%' || url_pattern || '%'))
              AND
          (@statuses::TEXT[] IS NULL OR o.status = ANY (@statuses))
              AND
          (@tags::TEXT[] IS NULL OR EXISTS (SELECT 1
                                            FROM unnest(@tags) AS tag
                                            WHERE tag = ANY (o.tags)))
              AND
          (
              (sqlc.narg(created_after)::TIMESTAMP IS NULL OR o.created_at >= sqlc.narg(created_after)) AND
              (sqlc.narg(created_before)::TIMESTAMP IS NULL OR o.created_at < sqlc.narg(created_before))
              )
              AND
          (
              (sqlc.narg(updated_after)::TIMESTAMP IS NULL OR o.updated_at >= sqlc.narg(updated_after)) AND
              (sqlc.narg(updated_before)::TIMESTAMP IS NULL OR o.updated_at < sqlc.narg(updated_before))
              )
          );