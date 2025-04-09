CREATE TABLE IF NOT EXISTS orders
(
    id             UUID      DEFAULT gen_random_uuid() NOT NULL,
    owner_id       VARCHAR(255)                        NOT NULL,
    price_amount   DECIMAL                             NOT NULL,
    price_currency VARCHAR(3)                          NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at     TIMESTAMP DEFAULT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS order_items
(
    order_id       UUID                                NOT NULL,
    product_id     UUID                                NOT NULL,
    price_amount   DECIMAL                             NOT NULL,
    price_currency VARCHAR(3)                          NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at     TIMESTAMP DEFAULT NULL,
    PRIMARY KEY (order_id, product_id),
    FOREIGN KEY (order_id) REFERENCES orders (id)
);

