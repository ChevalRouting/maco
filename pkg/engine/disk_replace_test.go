package engine

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/image"
	"github.com/m-vinc/maco/pkg/types"
)

func TestReplaceBootDisk(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img is required")
	}

	e := testEngine(t)
	m := &types.VMManifest{ID: uuid.NewString(), Name: "replace-test", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 1,
		Image: "old-image", BootOrder: []string{"net", "disk"},
		GuestSetup: &types.GuestSetup{Provisioner: types.ProvisionerCloudInit, Mode: types.GuestModeRaw, Raw: "#cloud-config\n"},
	}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(e.paths.VMDiskDir(m.ID), "disk.qcow2")
	writeDisk(t, path, 128)
	media := Media{ID: uuid.NewString(), Name: "fresh", Kind: "image", SizeGiB: 2}
	if err := os.MkdirAll(e.mediaDir(), 0o700); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(media)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(e.mediaDir(), media.ID+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	mediaPath := filepath.Join(e.mediaDir(), media.ID+".qcow2")
	if err := image.CreateVMDisk(context.Background(), "", mediaPath, 2); err != nil {
		t.Fatal(err)
	}

	if err := e.ManageDisk(context.Background(), m.ID, "vm.disk.replace", DiskParams{ID: "disk", ImageID: "invalid"}); err == nil {
		t.Fatal("invalid image accepted")
	}

	if contents, err := os.ReadFile(path); err != nil || len(contents) != 128 {
		t.Fatalf("old disk changed: %v", err)
	}

	if err := e.ManageDisk(context.Background(), m.ID, "vm.disk.replace", DiskParams{ID: "disk", ImageID: media.ID}); err != nil {
		t.Fatal(err)
	}

	saved, err := e.vms.Load(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	if saved.Image != "media:"+media.ID || saved.DiskSizeGiB != 2 || saved.BootOrder[0] != "disk" {
		t.Fatalf("incorrect replacement settings: %+v", saved)
	}

	if saved.GuestSetup.Raw != m.GuestSetup.Raw {
		t.Fatal("guest setup changed")
	}

	info, err := image.InspectStandalone(context.Background(), path)
	if err != nil || info.VirtualSize != 2<<30 {
		t.Fatalf("replacement disk invalid: %+v, %v", info, err)
	}

	if _, err := os.Stat(path + ".previous"); !os.IsNotExist(err) {
		t.Fatal("old disk was not cleaned up")
	}

	catalogID := "ubuntu-24.04-arm64"
	if err := image.CreateVMDisk(context.Background(), "", filepath.Join(e.paths.ImagesDir(), catalogID+".qcow2"), 3); err != nil {
		t.Fatal(err)
	}

	if err := e.ManageDisk(context.Background(), m.ID, "vm.disk.replace", DiskParams{ID: "disk", ImageID: catalogID}); err != nil {
		t.Fatal(err)
	}

	saved, err = e.vms.Load(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	if saved.Image != catalogID || saved.DiskSizeGiB != 3 || saved.GuestSetup.Raw != m.GuestSetup.Raw {
		t.Fatalf("incorrect catalog replacement: %+v", saved)
	}

	info, err = image.InspectStandalone(context.Background(), path)
	if err != nil || info.VirtualSize != 3<<30 {
		t.Fatalf("catalog replacement disk invalid: %+v, %v", info, err)
	}
}
