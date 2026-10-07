package vm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveDiskAttachAndGrow(t *testing.T) {
	qemu, err := exec.LookPath("qemu-system-aarch64")
	if err != nil {
		t.Skip("QEMU unavailable")
	}

	image, err := exec.LookPath("qemu-img")
	if err != nil {
		t.Skip("qemu-img unavailable")
	}

	dir := t.TempDir()
	driver := NewDriver(dir)
	id := "live"
	if err := os.MkdirAll(driver.vmRunDir(id), 0o700); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "data.qcow2")
	if output, err := exec.Command(image, "create", "-f", "qcow2", path, "1G").CombinedOutput(); err != nil {
		t.Fatalf("create: %v %s", err, output)
	}

	process := exec.Command(qemu, "-machine", "virt", "-accel", "tcg", "-cpu", "cortex-a72", "-m", "128", "-S", "-display", "none", "-nodefaults", "-device", "virtio-scsi-pci,id=scsi", "-qmp", "unix:"+driver.qmpPath(id)+",server=on,wait=off")
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() { _ = process.Process.Kill(); _ = process.Wait() }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(driver.qmpPath(id)); err == nil {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("QMP socket did not appear")
		}

		time.Sleep(10 * time.Millisecond)
	}

	if err := driver.AttachDisk(id, "f1658cbd-5e69-425f-a8ec-4321d072ae01", path); err != nil {
		t.Fatal(err)
	}

	if err := driver.GrowDisk(id, path, 2); err != nil {
		t.Fatal(err)
	}

	if err := driver.GrowDisk(id, path, 1); err == nil {
		t.Fatal("live shrink accepted")
	}

	client, err := dialQMP(driver.qmpPath(id), time.Second)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = client.close() }()

	data, err := client.execute("query-block")
	if err != nil {
		t.Fatal(err)
	}

	var devices []blockDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		t.Fatal(err)
	}

	if len(devices) != 1 || devices[0].Inserted == nil || devices[0].Inserted.Image.VirtualSize != 2*(1<<30) {
		t.Fatalf("wrong live capacity: %s", data)
	}
}
