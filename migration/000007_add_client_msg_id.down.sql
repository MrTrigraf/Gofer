ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_user_client_msg_id_key;
ALTER TABLE messages DROP COLUMN IF EXISTS client_msg_id;