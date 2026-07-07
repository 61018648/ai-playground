ALTER TABLE invite_codes
  ADD COLUMN IF NOT EXISTS amount numeric(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS used_by uuid REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS used_at timestamptz;

UPDATE invite_codes
SET max_uses = 1
WHERE max_uses <> 1;

CREATE INDEX IF NOT EXISTS idx_invite_codes_used_by ON invite_codes (used_by);
