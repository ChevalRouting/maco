package image

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func diskDetails(t *testing.T, path string) (int64, string) {
	t.Helper()
	out, err := exec.Command("qemu-img", "info", "--output=json", path).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect disk: %v %s", err, out)
	}

	var info struct {
		VirtualSize int64  `json:"virtual-size"`
		Backing     string `json:"backing-filename"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		t.Fatal(err)
	}

	return info.VirtualSize, info.Backing
}

func TestVMDiskCopiesAndExpandsImage(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img required")
	}

	if _, err := exec.LookPath("qemu-io"); err != nil {
		t.Skip("qemu-io required")
	}

	dir := t.TempDir()
	base := filepath.Join(dir, "base.qcow2")
	dst := filepath.Join(dir, "vm", "disk.qcow2")
	if out, err := exec.Command("qemu-img", "create", "-f", "qcow2", base, "1G").CombinedOutput(); err != nil {
		t.Fatalf("create base: %v %s", err, out)
	}

	if out, err := exec.Command("qemu-io", "-f", "qcow2", "-c", "write -P 90 0 4096", base).CombinedOutput(); err != nil {
		t.Fatalf("write base: %v %s", err, out)
	}

	if err := CreateVMDisk(context.Background(), base, dst, 3); err != nil {
		t.Fatal(err)
	}

	if size, backing := diskDetails(t, dst); size != 3*(1<<30) || backing != "" {
		t.Fatalf("disk size %d, backing %s", size, backing)
	}

	if size, _ := diskDetails(t, base); size != 1<<30 {
		t.Fatal("source image was resized")
	}

	if out, err := exec.Command("qemu-io", "-f", "qcow2", "-c", "read -P 90 0 4096", dst).CombinedOutput(); err != nil {
		t.Fatalf("image contents not preserved: %v %s", err, out)
	}

	if err := os.Remove(base); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("qemu-img", "check", dst).CombinedOutput(); err != nil {
		t.Fatalf("VM disk depends on source: %v %s", err, out)
	}

	if err := CreateVMDisk(context.Background(), base, dst, 4); err != nil {
		t.Fatal(err)
	}

	if size, _ := diskDetails(t, dst); size != 3*(1<<30) {
		t.Fatal("restart replaced existing disk")
	}
}

func TestVMDiskBlankAndMinimumCapacity(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img required")
	}

	dir := t.TempDir()
	base := filepath.Join(dir, "base.qcow2")
	if err := CreateVMDisk(context.Background(), "", base, 2); err != nil {
		t.Fatal(err)
	}

	if size, backing := diskDetails(t, base); size != 2*(1<<30) || backing != "" {
		t.Fatal("invalid blank disk")
	}

	dst := filepath.Join(dir, "small.qcow2")
	if err := CreateVMDisk(context.Background(), base, dst, 1); err == nil {
		t.Fatal("accepted capacity smaller than image")
	}

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("published rejected disk")
	}
}
