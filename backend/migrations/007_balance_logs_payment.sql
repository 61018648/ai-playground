CREATE TABLE IF NOT EXISTS balance_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  operator_id uuid REFERENCES users(id) ON DELETE SET NULL,
  change_type text NOT NULL,
  amount numeric(12,2) NOT NULL,
  balance_before numeric(12,2) NOT NULL,
  balance_after numeric(12,2) NOT NULL,
  note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_balance_logs_user_created_at ON balance_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_balance_logs_created_at ON balance_logs (created_at DESC);

INSERT INTO site_settings (key, value)
VALUES (
  'payment',
  '{"enabled":false,"provider":"epay","gatewayUrl":"","pid":"","key":"","notifyUrl":"","returnUrl":"","signType":"MD5"}'
)
ON CONFLICT (key) DO NOTHING;
