package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/types"
)

func writeDisk(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListDisks(t *testing.T) {
	e := testEngine(t)

	dataID := uuid.NewString()
	m := &types.VMManifest{
		ID: uuid.NewString(), Name: "alpha", Image: "img", CPUs: 1, MemoryMiB: 128,
		DiskSizeGiB: 10,
		Disks:       []types.VMDisk{{ID: dataID, Name: "extra", SizeGiB: 5}},
	}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	dir := e.paths.VMDiskDir(m.ID)
	writeDisk(t, filepath.Join(dir, "disk.qcow2"), 2048)
	writeDisk(t, filepath.Join(dir, dataID+".qcow2"), 1024)
	writeDisk(t, filepath.Join(dir, "seed.iso"), 512)
	strayID := uuid.NewString()
	writeDisk(t, filepath.Join(dir, strayID+".qcow2"), 256)

	ghost := e.paths.VMDiskDir("ghost")
	writeDisk(t, filepath.Join(ghost, "disk.qcow2"), 128)

	disks, err := e.ListDisks()
	if err != nil {
		t.Fatal(err)
	}

	byPath := make(map[string]DiskView, len(disks))
	for _, disk := range disks {
		byPath[disk.Path] = disk
	}

	if len(disks) != 4 {
		t.Fatalf("expected 4 disks with seed.iso ignored, got %d: %+v", len(disks), disks)
	}

	boot := byPath[filepath.Join(dir, "disk.qcow2")]
	if boot.Kind != "boot" || boot.Orphaned || boot.CapacityGiB != 10 || boot.VMName != "alpha" || boot.SizeBytes != 2048 {
		t.Fatalf("boot disk view wrong: %+v", boot)
	}

	data := byPath[filepath.Join(dir, dataID+".qcow2")]
	if data.Kind != "data" || data.Orphaned || data.CapacityGiB != 5 || data.Name != "extra" {
		t.Fatalf("data disk view wrong: %+v", data)
	}

	stray := byPath[filepath.Join(dir, strayID+".qcow2")]
	if !stray.Orphaned || stray.VMName != "alpha" {
		t.Fatalf("stray disk should be orphaned inside a known VM dir: %+v", stray)
	}

	ghostDisk := byPath[filepath.Join(ghost, "disk.qcow2")]
	if !ghostDisk.Orphaned || ghostDisk.VMName != "" || ghostDisk.Kind != "boot" {
		t.Fatalf("ghost disk should be orphaned with no VM: %+v", ghostDisk)
	}
}

func TestDiskFirstBootBadge(t *testing.T) {
	m := &types.VMManifest{Name: "vm", DiskSizeGiB: 10, Disks: []types.VMDisk{{ID: "second", Name: "Second", SizeGiB: 5}}, BootOrder: []string{"disk:second", "disk"}}
	primary := describeDisk(m, "vm", "disk.qcow2", "/disk", 0)
	second := describeDisk(m, "vm", "second.qcow2", "/second", 0)
	if primary.BootFirst || !second.BootFirst {
		t.Fatalf("incorrect priority: %+v %+v", primary, second)
	}

	m.ISOs = []string{"installer"}
	m.BootOrder = []string{"iso:installer", "disk:second", "disk"}
	if describeDisk(m, "vm", "second.qcow2", "/second", 0).BootFirst {
		t.Fatal("disk marked first when ISO boots first")
	}

	m.BootOrder = nil
	m.ISOs = nil
	if !describeDisk(m, "vm", "disk.qcow2", "/disk", 0).BootFirst {
		t.Fatal("default first disk not marked")
	}

	if describeDisk(nil, "vm", "disk.qcow2", "/disk", 0).BootFirst {
		t.Fatal("orphan marked first")
	}
}
