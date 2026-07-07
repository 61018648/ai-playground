UPDATE affiliate_profiles
SET code = upper(substr(encode(gen_random_bytes(6), 'hex'), 1, 8)),
    updated_at = now()
WHERE code LIKE 'STAR%' OR length(code) <> 8;
