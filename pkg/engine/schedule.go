package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/m-vinc/maco/pkg/types"
)

type ScheduleParams struct {
	Enabled       bool `json:"enabled" binding:"optional" extensions:"x-maco-description=Enable automatic backups for this VM."`
	IntervalHours int  `json:"interval_hours" binding:"optional" extensions:"x-maco-description=Hours between backups. Must be at least 1 when enabled." minimum:"0"`
	KeepLast      int  `json:"keep_last" binding:"optional" extensions:"x-maco-description=Number of newest backups retained. Zero disables this retention limit." minimum:"0"`
	MaxAgeDays    int  `json:"max_age_days" binding:"optional" extensions:"x-maco-description=Maximum backup age in days. Zero disables this retention limit." minimum:"0"`
}

func (e *Engine) GetSchedule(ctx context.Context, ref string) (*types.BackupSchedule, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}
	schedule, ok, err := database.GetBackupSchedule(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &types.BackupSchedule{VMID: m.ID, IntervalHours: 24}, nil
	}
	return &schedule, nil
}

func (e *Engine) SetSchedule(ctx context.Context, ref string, params ScheduleParams) (*types.BackupSchedule, error) {
	if params.Enabled && params.IntervalHours < 1 {
		return nil, fmt.Errorf("interval must be at least 1 hour")
	}
	if params.KeepLast < 0 || params.MaxAgeDays < 0 {
		return nil, fmt.Errorf("retention limits cannot be negative")
	}
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}
	schedule := types.BackupSchedule{
		VMID:          m.ID,
		Enabled:       params.Enabled,
		IntervalHours: params.IntervalHours,
		KeepLast:      params.KeepLast,
		MaxAgeDays:    params.MaxAgeDays,
	}
	if schedule.IntervalHours < 1 {
		schedule.IntervalHours = 24
	}
	if err := database.SetBackupSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	return e.GetSchedule(ctx, m.ID)
}

func (e *Engine) ListSchedules(ctx context.Context) ([]types.BackupSchedule, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}
	return database.ListBackupSchedules(ctx)
}

func (e *Engine) MarkScheduleRun(ctx context.Context, vmID string, at time.Time) error {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return err
	}
	return database.MarkBackupScheduleRun(ctx, vmID, at.Unix())
}

func (e *Engine) PruneBackups(ctx context.Context, ref string, keepLast, maxAgeDays int) (int, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return 0, err
	}
	backups, err := e.ListBackups(m.ID)
	if err != nil {
		return 0, err
	}

	remove := map[string]bool{}
	if keepLast > 0 && len(backups) > keepLast {
		for _, stale := range backups[keepLast:] {
			remove[stale.Timestamp] = true
		}
	}
	if maxAgeDays > 0 {
		cutoff := time.Now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)
		for _, backup := range backups {
			created := backupCreatedAt(backup)
			if !created.IsZero() && created.Before(cutoff) {
				remove[backup.Timestamp] = true
			}
		}
	}

	deleted := 0
	for timestamp := range remove {
		if err := e.DeleteBackup(m.ID, timestamp); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func backupCreatedAt(backup BackupInfo) time.Time {
	if backup.CreatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, backup.CreatedAt); err == nil {
			return parsed
		}
	}
	if len(backup.Timestamp) >= 15 {
		if parsed, err := time.Parse("20060102-150405", backup.Timestamp[:15]); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
