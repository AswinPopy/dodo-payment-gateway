ALTER TABLE businesses
ADD COLUMN webhook_url TEXT,
ADD COLUMN webhook_secret TEXT NOT NULL DEFAULT '';
