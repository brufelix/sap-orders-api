CREATE TABLE sap_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id   UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    order_number    VARCHAR(50) NOT NULL,
    rfc_function    VARCHAR(100) NOT NULL,
    xml_payload     TEXT NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 3,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ
);

CREATE INDEX idx_sap_outbox_status ON sap_outbox(status);
CREATE INDEX idx_sap_outbox_created_at ON sap_outbox(created_at);
