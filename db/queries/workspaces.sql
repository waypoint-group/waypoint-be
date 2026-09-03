-- name: CreateWorkspace :one
INSERT INTO workspaces (id, name)
VALUES ($1, $2)
RETURNING *;

-- name: SelectWorkspace :one
SELECT *
FROM workspaces
WHERE id = $1;

-- name: ListUserWorkspaces :many
SELECT workspaces.*
FROM workspaces
JOIN workspace_members
    ON workspace_members.workspace_id = workspaces.id
WHERE workspace_members.user_id = $1
ORDER BY workspaces.name;

-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;
