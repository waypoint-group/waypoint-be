CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL CONSTRAINT users_email_unique UNIQUE,
    user_name TEXT NOT NULL CONSTRAINT users_user_name_unique UNIQUE,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE channels (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT channels_workspace_name_unique UNIQUE (workspace_id, name),
    CONSTRAINT channels_workspace_id_id_unique UNIQUE (workspace_id, id)
);

CREATE TABLE channel_members (
    workspace_id UUID NOT NULL,
    channel_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (channel_id, user_id),
    -- Deleting a channel cascades to deleting all channel members.
    CONSTRAINT channel_members_workspace_channel_fkey
        FOREIGN KEY (workspace_id, channel_id)
        REFERENCES channels (workspace_id, id) ON DELETE CASCADE,
    -- Removing a workspace member cascades to removing channel membership
    -- (affecting all channels within that workspace).
    CONSTRAINT channel_members_workspace_member_fkey
        FOREIGN KEY (workspace_id, user_id)
        REFERENCES workspace_members (workspace_id, user_id) ON DELETE CASCADE
);

-- Support efficient cascading removal of a user's memberships within one workspace.
CREATE INDEX channel_members_workspace_user_idx
    ON channel_members (workspace_id, user_id);

CREATE TABLE messages (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    body TEXT NOT NULL CHECK (length(body) > 0),
    thread_root_id UUID REFERENCES messages(id) ON DELETE CASCADE
);

-- Supports message retrieval by channel and timestamp.
CREATE INDEX messages_channel_created_idx
    ON messages (channel_id, created_at DESC);

-- Supports thread lookups.
CREATE INDEX messages_thread_created_idx
    ON messages (thread_root_id, created_at)
    WHERE thread_root_id IS NOT NULL;
