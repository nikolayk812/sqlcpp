-- name: GetCart :many
SELECT product_id, price_amount, price_currency, created_at
FROM cart_items
WHERE owner_id = $1;

-- name: AddItem :exec
INSERT INTO cart_items (owner_id, product_id, price_amount, price_currency)
VALUES ($1, $2, $3, $4)
ON CONFLICT (owner_id, product_id) DO UPDATE
    SET price_amount = EXCLUDED.price_amount, price_currency = EXCLUDED.price_currency;

-- name: DeleteItem :execrows
DELETE FROM cart_items WHERE owner_id = $1 AND product_id = $2;