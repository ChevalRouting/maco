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
	KeepLast      int  `json:"keep_last" binding:"optional" extensions:"x-maco-description=Number of newest backups always retained. Zero disables this rule." minimum:"0"`
	KeepDaily     int  `json:"keep_daily" binding:"optional" extensions:"x-maco-description=Number of days for which the newest backup of the day is retained. Zero disables this rule." minimum:"0"`
	KeepWeekly    int  `json:"keep_weekly" binding:"optional" extensions:"x-maco-description=Number of ISO weeks for which the newest backup of the week is retained. Zero disables this rule." minimum:"0"`
	KeepMonthly   int  `json:"keep_monthly" binding:"optional" extensions:"x-maco-description=Number of months for which the newest backup of the month is retained. Zero disables this rule." minimum:"0"`
	KeepYearly    int  `json:"keep_yearly" binding:"optional" extensions:"x-maco-description=Number of years for which the newest backup of the year is retained. Zero disables this rule." minimum:"0"`
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

	if params.KeepLast < 0 || params.KeepDaily < 0 || params.KeepWeekly < 0 || params.KeepMonthly < 0 || params.KeepYearly < 0 {
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
		KeepDaily:     params.KeepDaily,
		KeepWeekly:    params.KeepWeekly,
		KeepMonthly:   params.KeepMonthly,
		KeepYearly:    params.KeepYearly,
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

func (e *Engine) PruneBackups(ctx context.Context, ref string) ([]string, error) {
	schedule, err := e.GetSchedule(ctx, ref)
	if err != nil {
		return nil, err
	}

	backups, err := e.ListBackups(schedule.VMID)
	if err != nil {
		return nil, err
	}

	deleted := make([]string, 0)
	for _, timestamp := range expiredBackups(backups, schedule.Retention()) {
		if err := e.DeleteBackup(schedule.VMID, timestamp); err != nil {
			return deleted, err
		}

		deleted = append(deleted, timestamp)
	}

	return deleted, nil
}
