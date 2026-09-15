-- name: CreateUserIdentity :one
INSERT INTO user_identities (id, user_id, issuer, subject)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SelectUserIdentity :one
SELECT *
FROM user_identities
WHERE id = $1;

-- name: SelectUserIdentityByUserID :one
SELECT *
FROM user_identities
WHERE user_id = $1;

-- name: ListUserIdentities :many
SELECT *
FROM user_identities;
