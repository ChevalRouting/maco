package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

func seedVMDisks(t *testing.T, e *Engine, m *types.VMManifest, names []string) {
	t.Helper()
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}
	dir := e.paths.VMDiskDir(m.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		out, err := exec.Command("qemu-img", "create", "-f", "qcow2", filepath.Join(dir, name), "1G").CombinedOutput()
		if err != nil {
			t.Fatalf("create %s: %v: %s", name, err, out)
		}
	}
}

func TestBackupLifecycle(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not installed")
	}
	e := testEngine(t)
	m := &types.VMManifest{ID: uuid.NewString(), Name: "gamma", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10}
	seedVMDisks(t, e, m, []string{"disk.qcow2"})

	res, err := e.BackupVM(context.Background(), m.Name)
	if err != nil {
		t.Fatal(err)
	}

	backups, err := e.ListBackups(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 || backups[0].Timestamp != res.Timestamp {
		t.Fatalf("expected one backup %s, got %+v", res.Timestamp, backups)
	}
	if backups[0].SizeBytes == 0 {
		t.Fatalf("expected non-zero backup size")
	}

	restored, err := e.RestoreVM(context.Background(), m.Name, res.Timestamp, true)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID == m.ID {
		t.Fatalf("restore as new must allocate a new id")
	}
	if _, err := os.Stat(filepath.Join(e.paths.VMDiskDir(restored.ID), "disk.qcow2")); err != nil {
		t.Fatalf("restored VM missing disk: %v", err)
	}

	if _, err := e.RestoreVM(context.Background(), m.Name, res.Timestamp, false); err != nil {
		t.Fatalf("in-place restore of stopped VM: %v", err)
	}

	if err := e.DeleteBackup(m.Name, "../escape"); err == nil {
		t.Fatalf("expected invalid backup id to be rejected")
	}
	if err := e.DeleteBackup(m.Name, res.Timestamp); err != nil {
		t.Fatal(err)
	}
	backups, err = e.ListBackups(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected no backups after delete, got %d", len(backups))
	}
}

func TestSnapshotLifecycleOffline(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not installed")
	}
	e := testEngine(t)
	m := &types.VMManifest{ID: uuid.NewString(), Name: "delta", Image: "img", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 10}
	seedVMDisks(t, e, m, []string{"disk.qcow2"})

	if _, err := e.CreateSnapshot(context.Background(), m.Name, SnapshotParams{Tag: "bad/name"}); err == nil {
		t.Fatalf("expected invalid snapshot tag to be rejected")
	}

	if _, err := e.CreateSnapshot(context.Background(), m.Name, SnapshotParams{Tag: "clean"}); err != nil {
		t.Fatal(err)
	}
	snapshots, err := e.ListSnapshots(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].Tag != "clean" {
		t.Fatalf("expected snapshot clean, got %+v", snapshots)
	}
	if snapshots[0].HasRAM {
		t.Fatalf("offline snapshot should not carry RAM state")
	}

	if err := e.RestoreSnapshot(context.Background(), m.Name, SnapshotParams{Tag: "clean"}); err != nil {
		t.Fatal(err)
	}
	if err := e.DeleteSnapshot(context.Background(), m.Name, SnapshotParams{Tag: "clean"}); err != nil {
		t.Fatal(err)
	}
	snapshots, err = e.ListSnapshots(m.Name)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected no snapshots after delete, got %d", len(snapshots))
	}
}
