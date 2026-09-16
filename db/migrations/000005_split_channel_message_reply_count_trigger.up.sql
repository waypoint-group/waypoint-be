DROP TRIGGER channel_message_reply_count ON channel_messages;
DROP FUNCTION update_channel_message_reply_count();

CREATE FUNCTION increment_channel_message_reply_count()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE channel_messages
    SET reply_count = reply_count + 1
    WHERE id = NEW.thread_root_id;

    RETURN NULL;
END;
$$;

CREATE FUNCTION decrement_channel_message_reply_count()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE channel_messages
    -- NOTE: `channel_messages` has a `reply_count >= 0` constraint.
    SET reply_count = reply_count - 1
    WHERE id = OLD.thread_root_id;

    RETURN NULL;
END;
$$;

CREATE TRIGGER channel_message_reply_count_insert
AFTER INSERT ON channel_messages
FOR EACH ROW
WHEN (NEW.thread_root_id IS NOT NULL)
EXECUTE FUNCTION increment_channel_message_reply_count();

CREATE TRIGGER channel_message_reply_count_delete
AFTER DELETE ON channel_messages
FOR EACH ROW
WHEN (OLD.thread_root_id IS NOT NULL)
EXECUTE FUNCTION decrement_channel_message_reply_count();
