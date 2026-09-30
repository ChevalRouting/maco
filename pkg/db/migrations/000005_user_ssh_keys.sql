-- +goose Up
CREATE TABLE IF NOT EXISTS user_ssh_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    public_key TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_user_ssh_keys_user ON user_ssh_keys(user_id);

-- +goose Down
DROP TABLE IF EXISTS user_ssh_keys;
