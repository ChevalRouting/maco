package engine

import (
	"fmt"
	"time"

	"github.com/m-vinc/maco/pkg/types"
)

type datedBackup struct {
	timestamp string
	created   time.Time
}

type retentionRule struct {
	keep   int
	period func(datedBackup) string
}

func expiredBackups(backups []BackupInfo, retention types.BackupRetention) []string {
	if !retention.Enabled() {
		return nil
	}

	dated := make([]datedBackup, 0, len(backups))
	kept := map[string]bool{}
	for _, backup := range backups {
		created := backupCreatedAt(backup)
		if created.IsZero() {
			kept[backup.Timestamp] = true
			continue
		}

		dated = append(dated, datedBackup{timestamp: backup.Timestamp, created: created.Local()})
	}

	rules := []retentionRule{
		{keep: retention.KeepLast, period: backupInstant},
		{keep: retention.KeepDaily, period: backupDay},
		{keep: retention.KeepWeekly, period: backupWeek},
		{keep: retention.KeepMonthly, period: backupMonth},
		{keep: retention.KeepYearly, period: backupYear},
	}
	for _, rule := range rules {
		keepPeriods(dated, kept, rule)
	}

	expired := make([]string, 0)
	for _, backup := range dated {
		if !kept[backup.timestamp] {
			expired = append(expired, backup.timestamp)
		}
	}

	return expired
}

func keepPeriods(backups []datedBackup, kept map[string]bool, rule retentionRule) {
	if rule.keep <= 0 {
		return
	}

	covered := map[string]bool{}
	for _, backup := range backups {
		if kept[backup.timestamp] {
			covered[rule.period(backup)] = true
		}
	}

	selected := map[string]bool{}
	for _, backup := range backups {
		period := rule.period(backup)
		if kept[backup.timestamp] || covered[period] || selected[period] {
			continue
		}

		if len(selected) >= rule.keep {
			return
		}

		selected[period] = true
		kept[backup.timestamp] = true
	}
}

func backupInstant(backup datedBackup) string {
	return backup.timestamp
}

func backupDay(backup datedBackup) string {
	return backup.created.Format("2006-01-02")
}

func backupWeek(backup datedBackup) string {
	year, week := backup.created.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

func backupMonth(backup datedBackup) string {
	return backup.created.Format("2006-01")
}

func backupYear(backup datedBackup) string {
	return backup.created.Format("2006")
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
