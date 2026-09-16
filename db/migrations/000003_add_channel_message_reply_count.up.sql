ALTER TABLE channel_messages
    ADD COLUMN reply_count BIGINT NOT NULL DEFAULT 0 CHECK (reply_count >= 0);

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
AFTER INSERT OR DELETE ON channel_messages
FOR EACH ROW EXECUTE FUNCTION update_channel_message_reply_count();
