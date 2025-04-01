CREATE TABLE IF NOT EXISTS cart_items
(
    owner_id       VARCHAR(255)                        NOT NULL,
    product_id     UUID                                NOT NULL,
    price_amount   DECIMAL                             NOT NULL,
    price_currency VARCHAR(3)                          NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (owner_id, product_id)
);

CREATE INDEX idx_cart_items_owner ON cart_items (owner_id);