ALTER TABLE users
  ADD COLUMN IF NOT EXISTS balance numeric(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS credits integer NOT NULL DEFAULT 0;

ALTER TABLE invite_codes
  ADD COLUMN IF NOT EXISTS note text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS created_by uuid REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_invite_codes_created_at ON invite_codes (created_at DESC);
