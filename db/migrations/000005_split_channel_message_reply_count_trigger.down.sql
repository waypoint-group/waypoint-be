DROP TRIGGER channel_message_reply_count_insert ON channel_messages;
DROP TRIGGER channel_message_reply_count_delete ON channel_messages;
DROP FUNCTION increment_channel_message_reply_count();
DROP FUNCTION decrement_channel_message_reply_count();

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
AFTER INSERT OR DELETE ON channel_messages
FOR EACH ROW EXECUTE FUNCTION update_channel_message_reply_count();
