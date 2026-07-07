CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  nickname text NOT NULL DEFAULT '',
  avatar_url text NOT NULL DEFAULT '',
  role text NOT NULL DEFAULT 'user',
  status text NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS verification_codes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email text NOT NULL,
  purpose text NOT NULL,
  code_hash text NOT NULL,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_verification_codes_email_purpose
  ON verification_codes (email, purpose, created_at DESC);

CREATE TABLE IF NOT EXISTS invite_codes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  max_uses integer NOT NULL DEFAULT 1,
  used_count integer NOT NULL DEFAULT 0,
  expires_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS apps (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code text NOT NULL UNIQUE,
  name text NOT NULL,
  category text NOT NULL,
  description text NOT NULL DEFAULT '',
  icon text NOT NULL DEFAULT '',
  icon_color text NOT NULL DEFAULT '',
  cover_url text NOT NULL DEFAULT '',
  prompt_template text NOT NULL DEFAULT '',
  input_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  output_schema jsonb NOT NULL DEFAULT '{}'::jsonb,
  owner_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  visibility text NOT NULL DEFAULT 'public',
  status text NOT NULL DEFAULT 'active',
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_apps_status_sort ON apps (status, sort_order, created_at DESC);

CREATE TABLE IF NOT EXISTS user_apps (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  app_id uuid NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  name text NOT NULL DEFAULT '',
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  pinned boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, app_id)
);

CREATE TABLE IF NOT EXISTS generation_jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  app_id uuid REFERENCES apps(id) ON DELETE SET NULL,
  prompt text NOT NULL,
  negative_prompt text NOT NULL DEFAULT '',
  params jsonb NOT NULL DEFAULT '{}'::jsonb,
  model text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'queued',
  progress integer NOT NULL DEFAULT 0,
  error_message text NOT NULL DEFAULT '',
  seed bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  started_at timestamptz,
  finished_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_generation_jobs_user_created
  ON generation_jobs (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_generation_jobs_status_created
  ON generation_jobs (status, created_at);

CREATE TABLE IF NOT EXISTS generation_assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id uuid NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
  kind text NOT NULL DEFAULT 'image',
  url text NOT NULL,
  thumbnail_url text NOT NULL DEFAULT '',
  width integer NOT NULL DEFAULT 0,
  height integer NOT NULL DEFAULT 0,
  mime_type text NOT NULL DEFAULT 'image/png',
  sort_order integer NOT NULL DEFAULT 0,
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS favorites (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  asset_id uuid NOT NULL REFERENCES generation_assets(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, asset_id)
);

CREATE TABLE IF NOT EXISTS conversations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  app_id uuid REFERENCES apps(id) ON DELETE SET NULL,
  title text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id uuid NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
  role text NOT NULL,
  content text NOT NULL,
  meta jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO apps (code, name, category, description, icon, icon_color, prompt_template, input_schema, sort_order)
VALUES
  (
    'creative-image',
    '创意图片生成',
    '绘画作图',
    '用于海报、封面、插画和创意图生成。',
    'i-lucide-sparkles',
    'bg-emerald-100 text-emerald-600',
    '{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"提示词","required":true},{"name":"size","type":"select","label":"尺寸","options":["1024x1024","1024x1536","1536x1024"]}]}',
    10
  ),
  (
    'ai-drawing',
    'AI 绘画作图',
    '绘画作图',
    '用于海报、封面、插画和创意图生成。',
    'i-lucide-pen-tool',
    'bg-slate-900 text-white',
    '{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"提示词","required":true},{"name":"style","type":"select","label":"风格","options":["写实","动漫","商业海报","摄影"]}]}',
    20
  ),
  (
    'ecommerce-visual',
    '电商视觉设计',
    '电商营销',
    '商品主图、详情页和营销素材一键生成。',
    'i-lucide-shopping-bag',
    'bg-red-100 text-red-600',
    '帮我设计一张电商商品图：{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"商品与卖点","required":true},{"name":"platform","type":"select","label":"平台","options":["淘宝","京东","抖音","小红书"]}]}',
    30
  ),
  (
    'poster-design',
    '海报设计',
    '平面设计',
    '活动宣传、品牌传播和社媒海报快速出图。',
    'i-lucide-image',
    'bg-cyan-100 text-cyan-700',
    '帮我设计一张活动宣传海报：{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"主题","required":true},{"name":"tone","type":"select","label":"调性","options":["高级","活泼","科技","节日"]}]}',
    40
  ),
  (
    'xiaohongshu-copy',
    '小红书爆款文案',
    '文本创作',
    '生成适合小红书发布的标题和正文。',
    'i-lucide-book-heart',
    'bg-rose-100 text-rose-600',
    '作为小红书运营专家，请围绕以下主题生成文案：{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"主题","required":true}]}',
    50
  ),
  (
    'writing-assistant',
    '全能写作助手',
    '工作助手',
    '提供写作灵感、文案润色和结构梳理。',
    'i-lucide-edit-3',
    'bg-yellow-100 text-yellow-700',
    '{{prompt}}',
    '{"fields":[{"name":"prompt","type":"textarea","label":"写作需求","required":true}]}',
    60
  )
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  category = EXCLUDED.category,
  description = EXCLUDED.description,
  icon = EXCLUDED.icon,
  icon_color = EXCLUDED.icon_color,
  prompt_template = EXCLUDED.prompt_template,
  input_schema = EXCLUDED.input_schema,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();
