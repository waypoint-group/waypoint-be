-- Enforce at most one owner per workspace, including direct database writes.
-- Atomic workspace creation and the service's removal rules preserve an owner.
CREATE UNIQUE INDEX workspace_members_single_owner_idx
    ON workspace_members (workspace_id)
    WHERE role = 'owner';
