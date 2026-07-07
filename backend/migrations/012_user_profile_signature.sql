ALTER TABLE users
  ADD COLUMN IF NOT EXISTS signature text NOT NULL DEFAULT '';

UPDATE affiliate_profiles
SET code = upper(substr(replace(user_id::text, '-', ''), 1, 8)),
    updated_at = now()
WHERE code LIKE 'ZX%';
