ALTER TABLE payment_orders
  ADD COLUMN IF NOT EXISTS cancelled_at timestamptz,
  ADD COLUMN IF NOT EXISTS expires_at timestamptz;

UPDATE payment_orders
SET expires_at = created_at + interval '15 minutes'
WHERE expires_at IS NULL;

ALTER TABLE payment_orders
  ALTER COLUMN expires_at SET DEFAULT (now() + interval '15 minutes'),
  ALTER COLUMN expires_at SET NOT NULL;
