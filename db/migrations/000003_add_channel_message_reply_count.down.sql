DROP TRIGGER channel_message_reply_count ON channel_messages;
DROP FUNCTION update_channel_message_reply_count();
ALTER TABLE channel_messages DROP COLUMN reply_count;
