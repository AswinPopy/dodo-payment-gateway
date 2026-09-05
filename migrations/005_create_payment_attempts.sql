CREATE TABLE payment_attempts (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    business_id UUID NOT NULL REFERENCES businesses(id),

    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,

    status TEXT NOT NULL DEFAULT 'PENDING',

    psp_transaction_id TEXT,
    failure_code TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT payment_attempts_amount_positive
        CHECK (amount > 0),

    CONSTRAINT payment_attempts_status_valid
        CHECK (
            status IN (
                'PENDING',
                'SUCCEEDED',
                'FAILED'
            )
        )
);

CREATE INDEX idx_payment_attempts_invoice_id
ON payment_attempts(invoice_id);

CREATE INDEX idx_payment_attempts_business_id
ON payment_attempts(business_id);

CREATE UNIQUE INDEX idx_one_pending_attempt_per_invoice
ON payment_attempts(invoice_id)
WHERE status = 'PENDING';