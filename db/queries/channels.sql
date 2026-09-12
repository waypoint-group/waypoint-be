-- name: CreateChannel :one
INSERT INTO channels (id, workspace_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SelectChannel :one
SELECT *
FROM channels
WHERE id = $1;

-- name: ListWorkspaceChannels :many
SELECT *
FROM channels
WHERE workspace_id = $1
ORDER BY name;

-- name: AddChannelMember :exec
INSERT INTO channel_members (channel_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteChannel :exec
DELETE FROM channels
WHERE id = $1;
