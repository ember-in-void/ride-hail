-- Seed data: default admin user for dev/demo environments
-- This migration is idempotent (ON CONFLICT DO NOTHING)

-- Password: admin123
-- Hash generated with: bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
-- Cost: 10 (bcrypt default)

INSERT INTO users (
    id,
    email,
    role,
    status,
    password_hash,
    attrs,
    created_at,
    updated_at
) VALUES (
    'admin-001',
    'admin@ridehail.com',
    'ADMIN',
    'ACTIVE',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- admin123
    '{"created_by": "migration", "description": "Default admin user"}'::jsonb,
    now(),
    now()
) ON CONFLICT (email) DO NOTHING;

-- Verify insertion (for logs)
DO $$
DECLARE
    admin_exists boolean;
BEGIN
    SELECT EXISTS(SELECT 1 FROM users WHERE email = 'admin@ridehail.com') INTO admin_exists;
    IF admin_exists THEN
        RAISE NOTICE 'Default admin user exists: admin@ridehail.com';
    ELSE
        RAISE WARNING 'Failed to create default admin user';
    END IF;
END $$;