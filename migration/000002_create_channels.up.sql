CREATE TABLE IF NOT EXISTS channels (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_name  VARCHAR(24) UNIQUE NOT NULL,
    created_by    UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ DEFAULT NOW()
);