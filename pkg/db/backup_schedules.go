package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/types"
)

func backupSchedule(row generated.BackupSchedule) types.BackupSchedule {
	return types.BackupSchedule{
		VMID:          row.VmID,
		Enabled:       row.Enabled != 0,
		IntervalHours: int(row.IntervalHours),
		KeepLast:      int(row.KeepLast),
		MaxAgeDays:    int(row.MaxAgeDays),
		LastRunAt:     row.LastRunAt,
	}
}

func (db *DB) SetBackupSchedule(ctx context.Context, schedule types.BackupSchedule) error {
	enabled := int64(0)
	if schedule.Enabled {
		enabled = 1
	}

	return db.queries.UpsertBackupSchedule(ctx, generated.UpsertBackupScheduleParams{
		VmID:          schedule.VMID,
		Enabled:       enabled,
		IntervalHours: int64(schedule.IntervalHours),
		KeepLast:      int64(schedule.KeepLast),
		MaxAgeDays:    int64(schedule.MaxAgeDays),
		LastRunAt:     schedule.LastRunAt,
	})
}

func (db *DB) GetBackupSchedule(ctx context.Context, vmID string) (types.BackupSchedule, bool, error) {
	row, err := db.queries.GetBackupSchedule(ctx, vmID)
	if errors.Is(err, sql.ErrNoRows) {
		return types.BackupSchedule{}, false, nil
	}

	if err != nil {
		return types.BackupSchedule{}, false, err
	}

	return backupSchedule(row), true, nil
}

func (db *DB) ListBackupSchedules(ctx context.Context) ([]types.BackupSchedule, error) {
	rows, err := db.queries.ListBackupSchedules(ctx)
	if err != nil {
		return nil, err
	}

	schedules := make([]types.BackupSchedule, 0, len(rows))
	for _, row := range rows {
		schedules = append(schedules, backupSchedule(row))
	}

	return schedules, nil
}

func (db *DB) MarkBackupScheduleRun(ctx context.Context, vmID string, at int64) error {
	return db.queries.MarkBackupScheduleRun(ctx, generated.MarkBackupScheduleRunParams{VmID: vmID, LastRunAt: at})
}

func (db *DB) DeleteBackupSchedule(ctx context.Context, vmID string) error {
	return db.queries.DeleteBackupSchedule(ctx, vmID)
}
