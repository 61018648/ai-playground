CREATE TABLE IF NOT EXISTS affiliate_profiles (
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  code text NOT NULL UNIQUE,
  level text NOT NULL DEFAULT '初级代理',
  commission_rate numeric(5,2) NOT NULL DEFAULT 20.00,
  visits integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS affiliate_referrals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  referrer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  referred_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (referred_user_id)
);

CREATE INDEX IF NOT EXISTS idx_affiliate_referrals_referrer_created_at
  ON affiliate_referrals (referrer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS affiliate_commissions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  referrer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  referred_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  payment_order_id uuid NOT NULL REFERENCES payment_orders(id) ON DELETE CASCADE,
  order_amount numeric(12,2) NOT NULL,
  product_type text NOT NULL,
  status text NOT NULL DEFAULT 'settled',
  commission_rate numeric(5,2) NOT NULL,
  commission_amount numeric(12,2) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (payment_order_id)
);

CREATE INDEX IF NOT EXISTS idx_affiliate_commissions_referrer_created_at
  ON affiliate_commissions (referrer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS affiliate_withdrawals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  amount numeric(12,2) NOT NULL,
  status text NOT NULL DEFAULT 'pending',
  note text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_affiliate_withdrawals_user_created_at
  ON affiliate_withdrawals (user_id, created_at DESC);

INSERT INTO affiliate_profiles (user_id, code)
SELECT id, upper(substr(encode(gen_random_bytes(6), 'hex'), 1, 8))
FROM users
WHERE deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING;
