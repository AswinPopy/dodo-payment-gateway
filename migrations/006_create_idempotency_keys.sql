CREATE TABLE idempotency_keys (
    id UUID PRIMARY KEY,

    business_id UUID NOT NULL REFERENCES businesses(id),

    idempotency_key TEXT NOT NULL,

    invoice_id UUID NOT NULL REFERENCES invoices(id),

    payment_attempt_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (business_id, idempotency_key)
);

CREATE INDEX idx_idempotency_keys_business_id
ON idempotency_keys(business_id);