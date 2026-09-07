ALTER TABLE channel_messages
    ADD COLUMN reply_count BIGINT NOT NULL DEFAULT 0 CHECK (reply_count >= 0);

UPDATE channel_messages AS root
SET reply_count = counts.reply_count
FROM (
    SELECT thread_root_id, COUNT(*) AS reply_count
    FROM channel_messages
    WHERE thread_root_id IS NOT NULL
    GROUP BY thread_root_id
) AS counts
WHERE root.id = counts.thread_root_id;

-- Update counts atomically in the same transaction as the reply change.
CREATE FUNCTION update_channel_message_reply_count() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE channel_messages
        SET reply_count = reply_count + 1
        WHERE id = NEW.thread_root_id;
    END IF;

    IF TG_OP = 'DELETE' THEN
        UPDATE channel_messages
        SET reply_count = reply_count - 1
        WHERE id = OLD.thread_root_id;
    END IF;

    RETURN NULL;
END;
$$;

CREATE TRIGGER channel_message_reply_count
-- UPDATE tag is ignored because that would indicate moving a reply 
-- from one root to another, which is not supported.
AFTER INSERT OR DELETE OF thread_root_id ON channel_messages
FOR EACH ROW EXECUTE FUNCTION update_channel_message_reply_count();
