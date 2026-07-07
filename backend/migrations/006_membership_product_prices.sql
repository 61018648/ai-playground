ALTER TABLE users
  ADD COLUMN IF NOT EXISTS membership_level text NOT NULL DEFAULT 'free';

ALTER TABLE apps
  ADD COLUMN IF NOT EXISTS price_free numeric(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS price_v1 numeric(12,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS price_v2 numeric(12,2) NOT NULL DEFAULT 0;

UPDATE users
SET membership_level = 'free'
WHERE membership_level = '';

UPDATE apps
SET price_free = 1.00,
    price_v1 = 0.80,
    price_v2 = 0.50
WHERE code IN ('ai-drawing', 'creative-image')
  AND price_free = 0
  AND price_v1 = 0
  AND price_v2 = 0;
