-- name: CreateMessage :one
INSERT INTO messages (id, author_id, channel_id, thread_root_id, body)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SelectMessage :one
SELECT *
FROM messages
WHERE id = $1;

-- name: ListChannelMessages :many
SELECT *
FROM messages
WHERE channel_id = $1
  AND thread_root_id IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListThreadMessages :many
SELECT *
FROM messages
WHERE thread_root_id = $1
ORDER BY created_at;

-- name: UpdateMessageBody :one
UPDATE messages
SET body = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1;
