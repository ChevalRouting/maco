package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
)

func TestHardwareUpdates(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	eng := New(paths)
	manifest, err := eng.CreateVM(CreateVMParams{Name: "hardware-test", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	cpus, memory := 4, 512
	if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{CPUs: &cpus, MemoryMiB: &memory}); err != nil {
		t.Fatal(err)
	}

	updated, err := eng.vms.Load(manifest.ID)
	if err != nil {
		t.Fatal(err)
	}

	if updated.CPUs != cpus || updated.MemoryMiB != memory || updated.DiskSizeGiB != 4 {
		t.Fatalf("incorrect hardware: %+v", updated)
	}

	invalid := 0
	if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{CPUs: &invalid}); err == nil {
		t.Fatal("invalid CPU count accepted")
	}

	run := paths.VMRunDir(manifest.ID)
	if err := os.MkdirAll(run, 0o700); err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.RemoveAll(run) }()

	pid := filepath.Join(run, "qemu.pid")
	if err := os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{CPUs: &cpus}); err == nil {
		t.Fatal("running VM hardware change accepted")
	}

	_ = os.Remove(pid)
	qemu, err := exec.LookPath("qemu-img")
	if err != nil {
		t.Skip("qemu-img required for disk resize check")
	}

	disk := filepath.Join(paths.VMDiskDir(manifest.ID), "disk.qcow2")
	if err := os.MkdirAll(filepath.Dir(disk), 0o700); err != nil {
		t.Fatal(err)
	}

	if output, err := exec.Command(qemu, "create", "-f", "qcow2", disk, "4G").CombinedOutput(); err != nil {
		t.Fatalf("create disk: %v: %s", err, output)
	}

	capacity := 6
	if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{DiskSizeGiB: &capacity}); err != nil {
		t.Fatal(err)
	}

	capacity = 5
	if err := eng.UpdateHardware(context.Background(), manifest.ID, UpdateHardwareParams{DiskSizeGiB: &capacity}); err == nil {
		t.Fatal("shrinking disk accepted")
	}
}
