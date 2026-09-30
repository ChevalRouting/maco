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

func TestBackupVM(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img not installed")
	}

	e := testEngine(t)

	dataID := uuid.NewString()
	m := &types.VMManifest{
		ID: uuid.NewString(), Name: "beta", Image: "img", CPUs: 1, MemoryMiB: 128,
		DiskSizeGiB: 10,
		Disks:       []types.VMDisk{{ID: dataID, Name: "extra", SizeGiB: 5}},
	}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	dir := e.paths.VMDiskDir(m.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"disk.qcow2", dataID + ".qcow2"} {
		out, err := exec.Command("qemu-img", "create", "-f", "qcow2", filepath.Join(dir, name), "1G").CombinedOutput()
		if err != nil {
			t.Fatalf("create %s: %v: %s", name, err, out)
		}
	}

	res, err := e.BackupVM(context.Background(), m.Name)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Disks) != 2 {
		t.Fatalf("expected 2 disks backed up, got %d: %+v", len(res.Disks), res.Disks)
	}

	for _, name := range []string{"disk.qcow2", dataID + ".qcow2", "manifest.yml"} {
		if _, err := exec.Command("test", "-f", filepath.Join(res.Path, name)).CombinedOutput(); err != nil {
			t.Fatalf("backup missing %s in %s", name, res.Path)
		}
	}

	if backing := qcow2Backing(filepath.Join(res.Path, "disk.qcow2")); backing != "" {
		t.Fatalf("backup disk should be self-contained, got backing %q", backing)
	}
}
