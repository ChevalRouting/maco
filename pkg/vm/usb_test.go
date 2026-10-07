package vm

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/usb"
)

type usbFixture struct {
	listener   net.Listener
	deleted    bool
	reject     bool
	seenDelete bool
	done       chan struct{}
}

func newUSBFixture(t *testing.T, reject bool) (*usbFixture, string) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "usb-qmp-")
	if err != nil {
		t.Fatal(err)
	}

	socket := filepath.Join(dir, "qmp.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}

	f := &usbFixture{listener: listener, reject: reject, done: make(chan struct{})}
	go f.run()
	t.Cleanup(func() { _ = listener.Close(); <-f.done; _ = os.RemoveAll(dir) })
	return f, socket
}

func (f *usbFixture) run() {
	defer close(f.done)
	conn, err := f.listener.Accept()
	if err != nil {
		return
	}

	defer func() { _ = conn.Close() }()
	enc, dec := json.NewEncoder(conn), json.NewDecoder(conn)
	_ = enc.Encode(map[string]any{"QMP": map[string]any{}})
	for {
		var command qmpCommand
		if dec.Decode(&command) != nil {
			return
		}

		args, _ := command.Arguments.(map[string]any)
		switch command.Execute {
		case "qmp_capabilities":
			_ = enc.Encode(map[string]any{"return": map[string]any{}})
		case "qom-list":
			result := []qomProperty{}
			if !f.deleted {
				result = append(result, qomProperty{Name: USBDeviceID("f1658cbd-5e69-425f-a8ec-4321d072ae01"), Type: "child<usb-host>"})
			}

			_ = enc.Encode(map[string]any{"return": result})
		case "qom-get":
			_ = enc.Encode(map[string]any{"return": true})
		case "device_del":
			f.seenDelete = true
			event := "DEVICE_DELETED"
			f.deleted = !f.reject
			if f.reject {
				event = "DEVICE_UNPLUG_GUEST_ERROR"
			}

			_ = enc.Encode(map[string]any{"event": event, "data": map[string]any{"device": args["id"]}})
			_ = enc.Encode(map[string]any{"return": map[string]any{}})
		default:
			_ = enc.Encode(map[string]any{"error": map[string]string{"class": "GenericError", "desc": "unexpected command"}})
		}
	}
}

func TestUSBDetachEventsBeforeResponse(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint(reject), func(t *testing.T) {
			_, socket := newUSBFixture(t, reject)
			err := DetachUSBAt(socket, USBDeviceID("f1658cbd-5e69-425f-a8ec-4321d072ae01"))
			if (err != nil) != reject {
				t.Fatalf("reject=%v error=%v", reject, err)
			}
		})
	}
}

func TestUSBRejectsEmulatedDeviceRemoval(t *testing.T) {
	if err := DetachUSBAt("/nonexistent", "usb-kbd"); err == nil {
		t.Fatal("accepted unmanaged device ID")
	}
}

func TestLiveUSBMissingCaptureAndDetach(t *testing.T) {
	qemu, err := exec.LookPath(QEMUBinary)
	if err != nil {
		t.Skip("QEMU unavailable")
	}

	dir, err := os.MkdirTemp("/tmp", "usb-live-")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()
	driver := NewDriver(dir)
	if err := os.MkdirAll(driver.vmRunDir("vm"), 0o700); err != nil {
		t.Fatal(err)
	}

	process := exec.Command(qemu, "-machine", "virt", "-accel", "tcg", "-S", "-display", "none", "-nodefaults", "-device", "qemu-xhci,id=maco-usb", "-qmp", "unix:"+driver.qmpPath("vm")+",server=on,wait=off")
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() { _ = process.Process.Kill(); _ = process.Wait() }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(driver.qmpPath("vm")); err == nil {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("QMP unavailable")
		}

		time.Sleep(10 * time.Millisecond)
	}

	id := USBDeviceID(uuid.NewString())
	err = driver.AttachUSB("vm", id, usb.Device{Bus: 255, Address: 127, Port: "7.7.7.7.7.7.7", VendorID: 65535, ProductID: 65535})
	if err == nil {
		t.Fatal("missing host device falsely reported captured")
	}

	state, err := ObserveUSB(driver.qmpPath("vm"), id)
	if err != nil || !state.Exists || state.Attached {
		t.Fatalf("unexpected uncaptured object: %+v %v", state, err)
	}

	if err := driver.DetachUSB("vm", id); err != nil {
		t.Fatal(err)
	}

	state, err = ObserveUSB(driver.qmpPath("vm"), id)
	if err != nil || state.Exists {
		t.Fatalf("device still present: %+v %v", state, err)
	}
}
