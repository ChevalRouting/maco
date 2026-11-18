package engine

import (
	"slices"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/types"
)

type retentionCase struct {
	name      string
	retention types.BackupRetention
	backups   []string
	expired   []string
}

func retentionBackups(t *testing.T, stamps []string) []BackupInfo {
	t.Helper()
	backups := make([]BackupInfo, 0, len(stamps))
	for _, stamp := range stamps {
		created, err := time.ParseInLocation("2006-01-02 15:04", stamp, time.Local)
		if err != nil {
			backups = append(backups, BackupInfo{Timestamp: stamp})
			continue
		}

		backups = append(backups, BackupInfo{Timestamp: stamp, CreatedAt: created.Format(time.RFC3339)})
	}

	return backups
}

func TestExpiredBackups(t *testing.T) {
	cases := []retentionCase{
		{
			name:    "no rules keeps everything",
			backups: []string{"2026-10-18 12:00", "2026-10-17 12:00"},
		},
		{
			name:      "keep last",
			retention: types.BackupRetention{KeepLast: 2},
			backups:   []string{"2026-10-18 12:00", "2026-10-18 08:00", "2026-10-17 12:00", "2026-10-16 12:00"},
			expired:   []string{"2026-10-17 12:00", "2026-10-16 12:00"},
		},
		{
			name:      "daily keeps the newest backup of each day",
			retention: types.BackupRetention{KeepDaily: 2},
			backups:   []string{"2026-10-18 12:00", "2026-10-18 08:00", "2026-10-17 20:00", "2026-10-17 10:00", "2026-10-16 12:00"},
			expired:   []string{"2026-10-18 08:00", "2026-10-17 10:00", "2026-10-16 12:00"},
		},
		{
			name:      "periods covered by earlier rules do not count",
			retention: types.BackupRetention{KeepLast: 1, KeepDaily: 2},
			backups:   []string{"2026-10-18 12:00", "2026-10-18 08:00", "2026-10-17 20:00", "2026-10-16 12:00", "2026-10-15 12:00"},
			expired:   []string{"2026-10-18 08:00", "2026-10-15 12:00"},
		},
		{
			name:      "weekly uses iso weeks",
			retention: types.BackupRetention{KeepWeekly: 2},
			backups:   []string{"2026-10-18 12:00", "2026-10-12 12:00", "2026-10-11 12:00", "2026-10-04 12:00"},
			expired:   []string{"2026-10-12 12:00", "2026-10-04 12:00"},
		},
		{
			name:      "monthly skips gaps",
			retention: types.BackupRetention{KeepMonthly: 3},
			backups:   []string{"2026-01-15 12:00", "2026-01-02 12:00", "2025-12-20 12:00", "2025-11-30 12:00", "2025-10-01 12:00"},
			expired:   []string{"2026-01-02 12:00", "2025-10-01 12:00"},
		},
		{
			name:      "yearly after monthly",
			retention: types.BackupRetention{KeepMonthly: 1, KeepYearly: 2},
			backups:   []string{"2026-03-01 12:00", "2026-02-01 12:00", "2025-06-01 12:00", "2024-06-01 12:00", "2023-06-01 12:00"},
			expired:   []string{"2026-02-01 12:00", "2023-06-01 12:00"},
		},
		{
			name:      "undated backups are kept",
			retention: types.BackupRetention{KeepLast: 1},
			backups:   []string{"2026-10-18 12:00", "undated", "2026-10-17 12:00"},
			expired:   []string{"2026-10-17 12:00"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkExpiredBackups(t, tc) })
	}
}

func checkExpiredBackups(t *testing.T, tc retentionCase) {
	got := expiredBackups(retentionBackups(t, tc.backups), tc.retention)
	if !slices.Equal(got, tc.expired) && (len(got) != 0 || len(tc.expired) != 0) {
		t.Fatalf("expired %v, want %v", got, tc.expired)
	}
}
