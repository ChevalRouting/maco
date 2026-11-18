-- +goose Up
ALTER TABLE backup_schedules ADD COLUMN keep_daily INTEGER NOT NULL DEFAULT 0;
ALTER TABLE backup_schedules ADD COLUMN keep_weekly INTEGER NOT NULL DEFAULT 0;
ALTER TABLE backup_schedules ADD COLUMN keep_monthly INTEGER NOT NULL DEFAULT 0;
ALTER TABLE backup_schedules ADD COLUMN keep_yearly INTEGER NOT NULL DEFAULT 0;
UPDATE backup_schedules SET keep_daily = max_age_days WHERE max_age_days > 0;
ALTER TABLE backup_schedules DROP COLUMN max_age_days;

-- +goose Down
ALTER TABLE backup_schedules ADD COLUMN max_age_days INTEGER NOT NULL DEFAULT 0;
UPDATE backup_schedules SET max_age_days = keep_daily;
ALTER TABLE backup_schedules DROP COLUMN keep_yearly;
ALTER TABLE backup_schedules DROP COLUMN keep_monthly;
ALTER TABLE backup_schedules DROP COLUMN keep_weekly;
ALTER TABLE backup_schedules DROP COLUMN keep_daily;
