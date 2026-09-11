CREATE TABLE user_identities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    auth_subject TEXT NOT NULL,
    auth_issuer TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (auth_issuer, auth_subject)
);

CREATE INDEX user_identities_user_id_idx
    ON user_identities (user_id);
