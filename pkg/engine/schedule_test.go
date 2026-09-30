package engine

import (
	"context"
	"os/exec"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

func TestBackupScheduleRoundTrip(t *testing.T) {
	e := testEngine(t)
	m := &types.VMManifest{ID: uuid.NewString(), Name: "epsilon", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	if _, err := e.SetSchedule(context.Background(), m.Name, ScheduleParams{Enabled: true, IntervalHours: 0}); err == nil {
		t.Fatalf("expected enabled schedule without interval to be rejected")
	}

	saved, err := e.SetSchedule(context.Background(), m.Name, ScheduleParams{Enabled: true, IntervalHours: 6, KeepLast: 3, MaxAgeDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Enabled || saved.IntervalHours != 6 || saved.KeepLast != 3 || saved.MaxAgeDays != 30 {
		t.Fatalf("unexpected saved schedule: %+v", saved)
	}

	got, err := e.GetSchedule(context.Background(), m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if got.KeepLast != 3 || got.IntervalHours != 6 {
		t.Fatalf("schedule not persisted: %+v", got)
	}
}

func TestPruneBackupsKeepLast(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not installed")
	}
	e := testEngine(t)
	m := &types.VMManifest{ID: uuid.NewString(), Name: "zeta", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10}
	seedVMDisks(t, e, m, []string{"disk.qcow2"})

	for i := 0; i < 3; i++ {
		if _, err := e.BackupVM(context.Background(), m.Name); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := e.PruneBackups(context.Background(), m.Name, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 backups pruned, got %d", deleted)
	}

	backups, err := e.ListBackups(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup kept, got %d", len(backups))
	}
}
