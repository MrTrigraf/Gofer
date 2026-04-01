CREATE TABLE IF NOT EXISTS messages (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content        TEXT NOT NULL,
    channel_id     UUID REFERENCES channels(id) ON DELETE CASCADE,
    direct_chat_id UUID REFERENCES direct_chats(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);