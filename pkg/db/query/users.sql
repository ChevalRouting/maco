-- name: CreateUser :exec
INSERT INTO users (id, username, password_hash, role, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetUserByUsername :one
SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?;

-- name: GetUserByID :one
SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?;

-- name: ListUsers :many
SELECT id, username, password_hash, role, created_at FROM users ORDER BY username;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ? WHERE username = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE username = ?;
