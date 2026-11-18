-- +goose Up
ALTER TABLE metric_samples ADD COLUMN cpu_basis TEXT NOT NULL DEFAULT 'allocated';
ALTER TABLE metric_samples ADD COLUMN counter_source TEXT NOT NULL DEFAULT 'route';

-- +goose Down
ALTER TABLE metric_samples DROP COLUMN counter_source;
ALTER TABLE metric_samples DROP COLUMN cpu_basis;
