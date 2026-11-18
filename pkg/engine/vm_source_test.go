package engine

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCreateVMExclusiveSource(t *testing.T) {
	e := testEngine(t)
	iso, err := e.CreateMedia(context.Background(), "installer.iso", 0, strings.NewReader("ISO"))
	if err != nil {
		t.Fatal(err)
	}

	defaults := CreateVMParams{Name: "iso-vm", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 2}
	both := defaults
	both.Image = "ubuntu-24.04-arm64"
	both.ISOs = []string{iso.ID}
	for _, params := range []CreateVMParams{defaults, both} {
		if _, err := e.CreateVM(params, nil); err == nil {
			t.Fatal("accepted non-exclusive source")
		}
	}

	defaults.ISOs = []string{iso.ID}
	defaults.BootOrder = []string{"iso:" + iso.ID, "disk"}
	m, err := e.CreateVM(defaults, nil)
	if err != nil {
		t.Fatal(err)
	}

	stored, err := e.vms.Resolve(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Image != "" || len(stored.ISOs) != 1 {
		t.Fatal("ISO creation added an image")
	}

	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img required")
	}

	path, err := e.prepareVMDisk(context.Background(), stored)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatal("ISO disk not created")
	}

	stored.Image = "media:missing"
	if again, err := e.prepareVMDisk(context.Background(), stored); err != nil || again != path {
		t.Fatalf("existing disk lost: %s %v", again, err)
	}
}
