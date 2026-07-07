UPDATE users
SET role = 'admin',
    status = 'active',
    updated_at = now()
WHERE email = 'codex-test@example.com';
