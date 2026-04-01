CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_name     VARCHAR(16) UNIQUE NOT NULL,
    password_hash VARCHAR(60) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);