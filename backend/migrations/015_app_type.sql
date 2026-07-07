ALTER TABLE apps
  ADD COLUMN IF NOT EXISTS app_type text NOT NULL DEFAULT 'image';

UPDATE apps
SET app_type = 'text'
WHERE code IN ('xiaohongshu-copy', 'writing-assistant')
  AND app_type = 'image';

UPDATE apps
SET app_type = 'image'
WHERE app_type = '';

CREATE INDEX IF NOT EXISTS idx_apps_type_status_sort
  ON apps (app_type, status, sort_order, created_at DESC);
