CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number    VARCHAR(50) NOT NULL UNIQUE,
    status          VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    created_by      VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    demand_code     VARCHAR(50) NOT NULL,
    description     TEXT NOT NULL,
    delivery_date   DATE NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, demand_code)
);

CREATE TABLE sap_sync_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id   UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    rfc_function    VARCHAR(100) NOT NULL,
    xml_request     TEXT NOT NULL,
    xml_response    TEXT,
    status          VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    error_message   TEXT,
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_delivery_date ON order_items(delivery_date);
CREATE INDEX idx_sap_sync_logs_order_item_id ON sap_sync_logs(order_item_id);
