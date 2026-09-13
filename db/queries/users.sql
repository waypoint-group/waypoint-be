-- name: CreateUser :one
INSERT INTO users (id, email, user_name, display_name)
VALUES ($1, $2, $3, $4)
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
ORDER BY id;
