ALTER TABLE api_keys
ADD COLUMN key_prefix TEXT NOT NULL DEFAULT '',
ADD COLUMN name TEXT,
ADD COLUMN last_used_at TIMESTAMPTZ;

CREATE INDEX idx_api_keys_key_prefix
ON api_keys(key_prefix);
