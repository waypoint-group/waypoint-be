-- Rename `messages` table to `channel_messages` since it is no
-- longer the only table for messages.
ALTER TABLE messages RENAME TO channel_messages;

-- Rename indexes to match the new table name.
ALTER INDEX messages_channel_created_idx RENAME TO channel_messages_channel_created_idx;
ALTER INDEX messages_thread_created_idx RENAME TO channel_messages_thread_created_idx;

-- Rename foreign key constraint to match the new table name.
ALTER TABLE channel_messages
    RENAME CONSTRAINT messages_channel_id_fkey TO channel_messages_channel_id_fkey;
ALTER TABLE channel_messages
    RENAME CONSTRAINT messages_author_id_fkey TO channel_messages_author_id_fkey;
ALTER TABLE channel_messages
    RENAME CONSTRAINT messages_thread_root_id_fkey TO channel_messages_thread_root_id_fkey;

CREATE TABLE direct_messages (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL CHECK (length(body) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Useful for quick direct message retrieval between users
-- in chronological order.
CREATE INDEX direct_messages_conversation_idx
ON direct_messages (
    LEAST(author_id, recipient_id),
    GREATEST(author_id, recipient_id),
    id DESC
);
