-- name: CreateUserSSHKey :exec
INSERT INTO user_ssh_keys (id, user_id, name, public_key, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListUserSSHKeysByUser :many
SELECT id, user_id, name, public_key, created_at
FROM user_ssh_keys
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: DeleteUserSSHKey :execrows
DELETE FROM user_ssh_keys WHERE id = ? AND user_id = ?;
