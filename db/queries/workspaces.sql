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
ORDER BY workspaces.name, workspaces.id;

-- name: AddWorkspaceMember :one
INSERT INTO workspace_members (workspace_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: SelectWorkspaceMember :one
SELECT *
FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: RemoveWorkspaceMember :exec
DELETE FROM workspace_members
WHERE workspace_id = $1 AND user_id = $2;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces
WHERE id = $1;

-- Serialize workspace operations through their shared parent row.
-- The lock lasts until the surrounding transaction commits or rolls back.
--
-- name: LockWorkspaceRow :one
SELECT * FROM workspaces WHERE id = $1 FOR UPDATE;

-- name: ListWorkspaceMembers :many
SELECT * FROM workspace_members WHERE workspace_id = $1 ORDER BY user_id;

-- Lock a bounded batch without waiting for active workspace operations or other cleaners.
--
-- NOTE: This statement executes in the following atomic operations:
-- 1) IDs of empty workspaces are selected from a snapshot S1
-- 2) The rows for said IDs (up to 100) are locked (skipping already locked rows)
--
-- Between these two operations, another transaction could have inserted a new user into
-- one of the previously empty workspaces. For this reason, the actual empty workspace
-- deletion happens in a separate statement in DeleteEmptyWorkspaces.
--
-- name: LockEmptyWorkspaces :many
SELECT id FROM workspaces
WHERE NOT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = workspaces.id)
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- Run after LockEmptyWorkspaces, in a separate statement to recheck membership.
--
-- name: DeleteEmptyWorkspaces :execrows
DELETE FROM workspaces
WHERE id = ANY(sqlc.arg(workspace_ids)::uuid[])
  AND NOT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = workspaces.id);
