CREATE TABLE IF NOT EXISTS payment_orders (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  trade_no text NOT NULL UNIQUE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider text NOT NULL DEFAULT 'epay',
  order_type text NOT NULL,
  plan_code text NOT NULL,
  plan_name text NOT NULL,
  amount numeric(12,2) NOT NULL,
  credits integer NOT NULL DEFAULT 0,
  membership_level text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'pending',
  pay_url text NOT NULL DEFAULT '',
  paid_at timestamptz,
  cancelled_at timestamptz,
  expires_at timestamptz NOT NULL DEFAULT (now() + interval '15 minutes'),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_orders_user_created_at ON payment_orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_orders_status_created_at ON payment_orders (status, created_at DESC);
