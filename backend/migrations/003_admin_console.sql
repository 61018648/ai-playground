CREATE TABLE IF NOT EXISTS site_settings (
  key text PRIMARY KEY,
  value jsonb NOT NULL DEFAULT '{}'::jsonb,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_providers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  provider text NOT NULL,
  base_url text NOT NULL DEFAULT '',
  api_key text NOT NULL DEFAULT '',
  model text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT false,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS login_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  email text NOT NULL DEFAULT '',
  success boolean NOT NULL DEFAULT false,
  ip text NOT NULL DEFAULT '',
  user_agent text NOT NULL DEFAULT '',
  message text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_login_logs_created_at ON login_logs (created_at DESC);

CREATE TABLE IF NOT EXISTS task_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id uuid REFERENCES generation_jobs(id) ON DELETE SET NULL,
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  action text NOT NULL,
  status text NOT NULL DEFAULT '',
  message text NOT NULL DEFAULT '',
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_task_logs_created_at ON task_logs (created_at DESC);

INSERT INTO site_settings (key, value)
VALUES
  (
    'seo',
    '{"siteName":"AI Playground","title":"AI Playground - AI 创作广场","description":"AI 创作平台，提供 AI 绘画、电商视觉、文案创作等一站式智能创作工具。","keywords":"AI生图,AI绘画,电商视觉,AI创作"}'
  ),
  (
    'auth',
    '{"allowRegister":true,"allowPasswordLogin":true,"requireEmailCode":true,"inviteOnly":false}'
  ),
  (
    'smtp',
    '{"host":"","port":587,"username":"","password":"","fromName":"AI Playground","fromEmail":"","secure":false}'
  )
ON CONFLICT (key) DO NOTHING;

INSERT INTO api_providers (name, provider, base_url, model, enabled, sort_order)
VALUES
  ('占位生图接口', 'placeholder', 'https://placehold.co', 'placeholder-v1', true, 10)
ON CONFLICT DO NOTHING;
