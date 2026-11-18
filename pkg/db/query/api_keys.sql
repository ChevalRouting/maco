-- name: CreateAPIKey :exec
INSERT INTO api_keys (id, user_id, name, token_hash, prefix, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListAPIKeysByUser :many
SELECT id, user_id, name, prefix, created_at, last_used_at, revoked_at
FROM api_keys
WHERE user_id = ? AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetActiveAPIKeyByHash :one
SELECT id, user_id, name, prefix, created_at, last_used_at, revoked_at
FROM api_keys
WHERE token_hash = ? AND revoked_at IS NULL;

-- name: RevokeAPIKey :execrows
UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used_at = ? WHERE id = ?;
