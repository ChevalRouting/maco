-- +goose Up
CREATE TABLE IF NOT EXISTS vm_state (
    id TEXT PRIMARY KEY,
    phase TEXT NOT NULL DEFAULT 'stopped',
    pid INTEGER NOT NULL DEFAULT 0,
    boot_time INTEGER NOT NULL DEFAULT 0,
    seen_at INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS vm_state;
