CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    business_id UUID NOT NULL REFERENCES businesses(id),
    customer_id UUID NOT NULL REFERENCES customers(id),

    currency TEXT NOT NULL,
    amount BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'DRAFT',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT invoices_amount_positive
        CHECK (amount > 0),

    CONSTRAINT invoices_status_valid
        CHECK (
            status IN (
                'DRAFT',
                'OPEN',
                'PAID',
                'VOID',
                'UNCOLLECTIBLE'
            )
        )
);

CREATE INDEX idx_invoices_business_id
ON invoices(business_id);

CREATE INDEX idx_invoices_customer_id
ON invoices(customer_id);