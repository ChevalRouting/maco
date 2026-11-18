-- +goose Up
CREATE TABLE goqite (
    id TEXT PRIMARY KEY DEFAULT ('m_' || lower(hex(randomblob(16)))),
    created TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    queue TEXT NOT NULL,
    body BLOB NOT NULL,
    timeout TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
    received INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0
) STRICT;

-- +goose StatementBegin
CREATE TRIGGER goqite_updated_timestamp AFTER UPDATE ON goqite BEGIN
    UPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;
-- +goose StatementEnd

CREATE INDEX goqite_queue_priority_created_idx ON goqite (queue, priority DESC, created);

CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,
    target TEXT NOT NULL,
    state TEXT NOT NULL,
    private INTEGER NOT NULL DEFAULT 0,
    message_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER,
    error TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL DEFAULT ''
);
CREATE INDEX jobs_private_created ON jobs(private, created_at);
CREATE INDEX jobs_state ON jobs(state);

CREATE TABLE job_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    message TEXT NOT NULL
);
CREATE INDEX job_logs_job ON job_logs(job_id, id);

CREATE TABLE job_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    payload TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX job_events_created ON job_events(created_at);

CREATE TABLE job_claims (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE job_claims;
DROP TABLE job_events;
DROP TABLE job_logs;
DROP TABLE jobs;
DROP TABLE goqite;
