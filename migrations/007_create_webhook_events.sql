CREATE TABLE webhook_events (
    id UUID PRIMARY KEY,
    event_id TEXT NOT NULL UNIQUE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT webhook_events_status_valid CHECK (
        status IN ('PENDING', 'PROCESSED', 'FAILED')
    )
);

CREATE INDEX idx_webhook_events_status
ON webhook_events(status);

CREATE INDEX idx_webhook_events_created_at
ON webhook_events(created_at);