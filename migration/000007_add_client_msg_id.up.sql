-- client_msg_id: идемпотентный ключ, который клиент генерит ОДИН раз
-- на сообщение и переиспользует при ретраях (Ctrl+R). UNIQUE не даёт
-- повторному INSERT создать дубль. NULLable — старые сообщения и
-- любой INSERT без ключа остаются валидными (в Postgres несколько
-- NULL не считаются нарушением UNIQUE).
ALTER TABLE messages ADD COLUMN client_msg_id UUID;

ALTER TABLE messages
    ADD CONSTRAINT messages_user_client_msg_id_key
    UNIQUE (user_id, client_msg_id);