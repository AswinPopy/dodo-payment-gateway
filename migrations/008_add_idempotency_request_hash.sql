ALTER TABLE idempotency_keys
ADD COLUMN request_hash TEXT NOT NULL DEFAULT '';
