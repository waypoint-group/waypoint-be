-- name: CreateUser :one
INSERT INTO users (id, email, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO NOTHING
RETURNING *;

-- name: SelectUser :one
SELECT *
FROM users
WHERE id = $1;

-- name: SelectUserByIdentity :one
SELECT u.*
FROM users AS u
JOIN user_identities AS i ON u.id = i.user_id
WHERE i.auth_subject = $1 AND i.auth_issuer = $2;

-- name: ListUsers :many
SELECT *
FROM users
ORDER BY display_name;
