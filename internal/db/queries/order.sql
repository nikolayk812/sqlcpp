-- name: GetOrder :one
SELECT id,
       owner_id,
       created_at,
       updated_at,
       url,
       status,
       tags,
       payload,
       payloadb
FROM orders
WHERE id = $1;

-- name: InsertOrder :one
INSERT INTO orders (owner_id, url, tags, payload, payloadb)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: GetOrderItems :many
SELECT product_id, price_amount, price_currency, created_at
FROM order_items
WHERE order_id = $1;

-- name: InsertOrderItem :exec
INSERT INTO order_items (order_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4);

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
       oi.product_id,
       oi.price_amount,
       oi.price_currency
FROM orders o
         JOIN order_items oi ON o.id = oi.order_id
WHERE o.id = $1;

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
       oi.product_id,
       oi.price_amount,
       oi.price_currency
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
                                            WHERE tag = ANY (@tags)))
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