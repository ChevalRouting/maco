-- name: UpsertBackupSchedule :exec
INSERT INTO backup_schedules (vm_id, enabled, interval_hours, keep_last, keep_daily, keep_weekly, keep_monthly, keep_yearly, last_run_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(vm_id) DO UPDATE SET
    enabled = excluded.enabled,
    interval_hours = excluded.interval_hours,
    keep_last = excluded.keep_last,
    keep_daily = excluded.keep_daily,
    keep_weekly = excluded.keep_weekly,
    keep_monthly = excluded.keep_monthly,
    keep_yearly = excluded.keep_yearly;

-- name: GetBackupSchedule :one
SELECT vm_id, enabled, interval_hours, keep_last, last_run_at, keep_daily, keep_weekly, keep_monthly, keep_yearly FROM backup_schedules WHERE vm_id = ?;

-- name: ListBackupSchedules :many
SELECT vm_id, enabled, interval_hours, keep_last, last_run_at, keep_daily, keep_weekly, keep_monthly, keep_yearly FROM backup_schedules;

-- name: MarkBackupScheduleRun :exec
UPDATE backup_schedules SET last_run_at = ? WHERE vm_id = ?;

-- name: DeleteBackupSchedule :exec
DELETE FROM backup_schedules WHERE vm_id = ?;
