-- name: CreateChannelMessage :one
INSERT INTO channel_messages (id, author_id, channel_id, body, thread_root_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SelectChannelMessage :one
SELECT *
FROM channel_messages
WHERE id = $1;

-- name: ListChannelMessages :many
SELECT *
FROM channel_messages
WHERE channel_id = $1
  AND thread_root_id IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListChannelThreadMessages :many
SELECT *
FROM channel_messages
WHERE thread_root_id = $1
ORDER BY created_at;

-- name: UpdateChannelMessageBody :one
UPDATE channel_messages
SET body = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteChannelMessage :execrows
DELETE FROM channel_messages
WHERE id = $1;
