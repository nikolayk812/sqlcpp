ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS url      TEXT,
    ADD COLUMN IF NOT EXISTS tags     TEXT[],
    ADD COLUMN IF NOT EXISTS status   TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS payload  JSON NOT NULL DEFAULT '{}'::JSON,
    ADD COLUMN IF NOT EXISTS payloadb JSONB;

ALTER TABLE orders
    ADD CONSTRAINT order_status_check
        CHECK (status IN ('pending', 'shipped', 'delivered', 'canceled'));
