ALTER TABLE apps
  ADD COLUMN IF NOT EXISTS provider_id uuid REFERENCES api_providers(id) ON DELETE SET NULL;

ALTER TABLE conversations
  ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'draw';

UPDATE conversations
SET kind = 'draw'
WHERE kind = '';

INSERT INTO api_providers (name, category, provider, base_url, model, enabled, sort_order)
VALUES ('智能助手接口', 'assistant_chat', 'openai', '', 'gpt-5.5', false, 20)
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_conversations_user_kind_updated
  ON conversations (user_id, kind, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_apps_provider_id
  ON apps (provider_id);
