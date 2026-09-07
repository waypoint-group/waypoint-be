DROP INDEX direct_messages_participants_created_idx;
DROP TABLE direct_messages;

ALTER TABLE channel_messages
    RENAME CONSTRAINT channel_messages_thread_root_id_fkey
    TO messages_thread_root_id_fkey;

ALTER TABLE channel_messages
    RENAME CONSTRAINT channel_messages_author_id_fkey
    TO messages_author_id_fkey;

ALTER TABLE channel_messages
    RENAME CONSTRAINT channel_messages_channel_id_fkey
    TO messages_channel_id_fkey;

ALTER INDEX channel_messages_thread_created_idx
    RENAME TO messages_thread_created_idx;

ALTER INDEX channel_messages_channel_created_idx
    RENAME TO messages_channel_created_idx;

ALTER TABLE channel_messages RENAME TO messages;
