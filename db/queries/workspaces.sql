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
ORDER BY workspaces.id;

-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SelectWorkspaceMember :one
SELECT *
FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: RemoveWorkspaceMember :execrows
DELETE FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteWorkspace :execrows
DELETE FROM workspaces
WHERE id = $1;

-- DeleteWorkspacesOwnedByUser removes all workspaces where the user is the owner.
-- Lock the user row first to exclude concurrent ownership creation during account deletion.
-- name: DeleteWorkspacesOwnedByUser :exec
DELETE FROM workspaces AS w
USING workspace_members AS m
WHERE m.workspace_id = w.id
  AND m.user_id = $1
  AND m.role = 'owner';

-- name: ListWorkspaceMembers :many
SELECT * FROM workspace_members WHERE workspace_id = $1 ORDER BY user_id;

-- Serialize workspace operations through their shared parent row.
-- The lock lasts until the surrounding transaction commits or rolls back.
--
-- name: LockWorkspaceRow :one
SELECT * FROM workspaces WHERE id = $1 FOR UPDATE;
