-- name: CreateDirectMessage :one
INSERT INTO direct_messages (id, author_id, recipient_id, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SelectDirectMessage :one
SELECT *
FROM direct_messages
WHERE id = $1;

-- name: ListDirectMessagesBetween :many
SELECT *
FROM direct_messages
WHERE LEAST(author_id, recipient_id)
        = LEAST(sqlc.arg('user_a'), sqlc.arg('user_b'))
    AND GREATEST(author_id, recipient_id)
        = GREATEST(sqlc.arg('user_a'), sqlc.arg('user_b'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateDirectMessageBody :one
UPDATE direct_messages
SET body = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDirectMessage :execrows
DELETE FROM direct_messages
WHERE id = $1;
