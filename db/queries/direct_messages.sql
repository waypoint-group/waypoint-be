-- name: CreateDirectMessage :one
INSERT INTO direct_messages (id, author_id, recipient_id, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SelectDirectMessage :one
SELECT *
FROM direct_messages
WHERE id = $1;

-- name: ListDirectMessages :many
SELECT *
FROM direct_messages
WHERE LEAST(author_id, recipient_id) = LEAST($1, $2)
  AND GREATEST(author_id, recipient_id) = GREATEST($1, $2)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: UpdateDirectMessageBody :one
UPDATE direct_messages
SET body = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDirectMessage :execrows
DELETE FROM direct_messages
WHERE id = $1;
