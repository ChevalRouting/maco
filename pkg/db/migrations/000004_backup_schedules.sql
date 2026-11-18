-- +goose Up
CREATE TABLE IF NOT EXISTS backup_schedules (
    vm_id TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    interval_hours INTEGER NOT NULL DEFAULT 24,
    keep_last INTEGER NOT NULL DEFAULT 0,
    max_age_days INTEGER NOT NULL DEFAULT 0,
    last_run_at INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS backup_schedules;
