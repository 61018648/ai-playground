ALTER TABLE api_providers
  ADD COLUMN IF NOT EXISTS category text NOT NULL DEFAULT 'general';

UPDATE api_providers
SET category = 'general'
WHERE category = '';

CREATE INDEX IF NOT EXISTS idx_api_providers_category_enabled
  ON api_providers (category, enabled, sort_order, created_at DESC);
