-- +goose Up
CREATE TABLE metric_samples (
    scope TEXT NOT NULL,
    target TEXT NOT NULL,
    ts INTEGER NOT NULL,
    cpu_percent REAL,
    memory_bytes INTEGER NOT NULL,
    memory_total INTEGER NOT NULL,
    rx_bytes INTEGER NOT NULL,
    tx_bytes INTEGER NOT NULL,
    rx_bytes_ps REAL,
    tx_bytes_ps REAL,
    PRIMARY KEY (scope, target, ts)
);
CREATE INDEX metric_samples_ts ON metric_samples(ts);

-- +goose Down
DROP TABLE metric_samples;
