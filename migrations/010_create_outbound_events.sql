CREATE TABLE outbound_events (
    id UUID PRIMARY KEY,
    business_id UUID NOT NULL REFERENCES businesses(id),
    source_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    attempt_count INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,

    CONSTRAINT outbound_events_status_valid CHECK (
        status IN ('PENDING', 'DELIVERING', 'DELIVERED', 'FAILED')
    ),

    CONSTRAINT outbound_events_source_unique
        UNIQUE (event_type, source_id)
);

CREATE INDEX idx_outbound_events_due
ON outbound_events (next_retry_at)
WHERE status = 'PENDING';

CREATE INDEX idx_outbound_events_business_created
ON outbound_events (business_id, created_at, id);
