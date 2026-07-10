UPDATE site_settings
SET value = jsonb_set(
    jsonb_set(
      jsonb_set(value, '{siteName}', to_jsonb('AI Playground'::text), true),
      '{title}', to_jsonb('AI Playground - AI 创作广场'::text), true
    ),
    '{description}', to_jsonb('AI 创作平台，提供 AI 绘画、电商视觉、文案创作等一站式智能创作工具。'::text), true
  ),
  updated_at = now()
WHERE key = 'seo'
  AND value = '{"siteName":"摘星AI","title":"摘星AI - AI 创作广场","description":"摘星AI 创作平台，提供 AI 绘画、电商视觉、文案创作等一站式智能创作工具。","keywords":"AI生图,AI绘画,电商视觉,AI创作"}'::jsonb;

UPDATE site_settings
SET value = jsonb_set(value, '{fromName}', to_jsonb('AI Playground'::text), true),
  updated_at = now()
WHERE key = 'smtp'
  AND value = '{"host":"","port":587,"username":"","password":"","fromName":"摘星AI","fromEmail":"","secure":false}'::jsonb;
