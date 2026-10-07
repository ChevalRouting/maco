package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/usb"
)

type engineUSBQMP struct {
	listener net.Listener
	objects  map[string]bool
	done     chan struct{}
}
type engineUSBCommand struct {
	Execute   string         `json:"execute"`
	Arguments map[string]any `json:"arguments"`
}
type engineUSBProperty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (f *engineUSBQMP) run() {
	defer close(f.done)
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}

		f.serve(conn)
		_ = conn.Close()
	}
}

func (f *engineUSBQMP) serve(conn net.Conn) {
	enc, dec := json.NewEncoder(conn), json.NewDecoder(conn)
	_ = enc.Encode(map[string]any{"QMP": map[string]any{}})
	for {
		var cmd engineUSBCommand
		if dec.Decode(&cmd) != nil {
			return
		}

		var result any = map[string]any{}
		switch cmd.Execute {
		case "device_add":
			f.objects[cmd.Arguments["id"].(string)] = true
		case "device_del":
			delete(f.objects, cmd.Arguments["id"].(string))
		case "qom-list":
			props := []engineUSBProperty{}
			for id := range f.objects {
				props = append(props, engineUSBProperty{id, "child<usb-host>"})
			}

			result = props
		case "qom-get":
			result = true
		}

		if enc.Encode(map[string]any{"return": result}) != nil {
			return
		}
	}
}

func usbTestEngine(t *testing.T, registry string) (*Engine, string, usb.Device) {
	t.Helper()
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	e := New(paths)
	e.usbClaimsDir = filepath.Join(registry, "usb")
	device := usb.Device{ID: strings.Repeat("a", 24), Fingerprint: strings.Repeat("b", 64), VendorID: 0x1050, ProductID: 0x0407, Bus: 1, Address: 2, Port: "1.2", Serial: "test", State: "available"}
	e.usbDevices = func() ([]usb.Device, error) { return []usb.Device{device}, nil }
	m, err := e.CreateVM(CreateVMParams{Name: "usb-test", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	run := paths.VMRunDir(m.ID)
	if err := os.MkdirAll(run, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(run, "qemu.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("unix", filepath.Join(run, "qmp.sock"))
	if err != nil {
		t.Fatal(err)
	}

	fixture := &engineUSBQMP{listener: listener, objects: map[string]bool{}, done: make(chan struct{})}
	go fixture.run()
	t.Cleanup(func() { _ = listener.Close(); <-fixture.done; _ = os.RemoveAll(paths.RunDir()) })
	return e, m.ID, device
}

func TestUSBExclusiveOwnershipAndRestart(t *testing.T) {
	registry := t.TempDir()
	e1, id1, device := usbTestEngine(t, registry)
	e2, id2, _ := usbTestEngine(t, registry)
	params := USBParams{DeviceID: device.ID, Fingerprint: device.Fingerprint}
	ctx := context.Background()
	var wg sync.WaitGroup
	var results [2]string
	var failures [2]error
	wg.Add(2)
	go func() { defer wg.Done(); results[0], failures[0] = e1.AttachUSB(ctx, id1, params) }()
	go func() { defer wg.Done(); results[1], failures[1] = e2.AttachUSB(ctx, id2, params) }()
	wg.Wait()
	if (failures[0] == nil) == (failures[1] == nil) {
		t.Fatalf("expected one winner: %v", failures)
	}

	winner, loser, vmID, otherID, attachment := e1, e2, id1, id2, results[0]
	if failures[0] != nil {
		winner, loser, vmID, otherID, attachment = e2, e1, id2, id1, results[1]
	}

	if err := loser.DetachUSB(ctx, otherID, attachment); err == nil {
		t.Fatal("wrong VM detached device")
	}

	restarted := New(winner.paths)
	restarted.usbClaimsDir = winner.usbClaimsDir
	restarted.usbDevices = winner.usbDevices
	list, err := restarted.ListVMUSB(ctx, vmID)
	if err != nil || len(list) != 1 || list[0].State != "attached" {
		t.Fatalf("lost live assignment on backend restart: %+v %v", list, err)
	}

	if err := restarted.DetachUSB(ctx, vmID, attachment); err != nil {
		t.Fatal(err)
	}

	if _, err := loser.AttachUSB(ctx, otherID, params); err != nil {
		t.Fatalf("released device unavailable: %v", err)
	}
}

func TestUSBStaleSelectionAndStoppedVM(t *testing.T) {
	e, id, d := usbTestEngine(t, t.TempDir())
	params := USBParams{DeviceID: d.ID, Fingerprint: strings.Repeat("c", 64)}
	if _, err := e.AttachUSB(context.Background(), id, params); err == nil {
		t.Fatal("stale fingerprint accepted")
	}

	if err := os.Remove(filepath.Join(e.paths.VMRunDir(id), "qemu.pid")); err != nil {
		t.Fatal(err)
	}

	params.Fingerprint = d.Fingerprint
	if _, err := e.AttachUSB(context.Background(), id, params); err == nil {
		t.Fatal("attached to stopped VM")
	}
}

func TestUnassignUnpluggedDevice(t *testing.T) {
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	e := New(paths)
	e.usbClaimsDir = filepath.Join(t.TempDir(), "usb")
	e.usbDevices = func() ([]usb.Device, error) { return []usb.Device{}, nil }
	m, err := e.CreateVM(CreateVMParams{Name: "usb-unplug", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(paths.VMRunDir(m.ID), 0o700); err != nil {
		t.Fatal(err)
	}

	m.USB = []types.VMUSBAssignment{{VendorID: 0x10c4, ProductID: 0xea60, Serial: "0001"}}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	key := AssignmentKey(m.USB[0])
	if err := e.UnassignUSB(context.Background(), m.ID, key); err != nil {
		t.Fatalf("unassign %q for an unplugged device: %v", key, err)
	}

	reloaded, err := e.vms.Load(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(reloaded.USB) != 0 {
		t.Fatalf("assignment not removed: %+v", reloaded.USB)
	}
}

func TestUSBUncertainClaimRetained(t *testing.T) {
	e, id, d := usbTestEngine(t, t.TempDir())
	ctx := context.Background()
	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		t.Fatal(err)
	}

	run := t.TempDir()
	if err := os.WriteFile(filepath.Join(run, "qemu.pid"), []byte(fmt.Sprint(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	generation, err := usbGeneration(run)
	if err != nil {
		t.Fatal(err)
	}

	r.claims = []usbClaim{{Attachment: USBAttachment{ID: "f1658cbd-5e69-425f-a8ec-4321d072ae01", VMID: "other", VMName: "other", Device: d}, QEMUID: "maco-usb-f1658cbd-5e69-425f-a8ec-4321d072ae01", RunDir: run, PID: os.Getpid(), Generation: generation}}
	if err := r.save(); err != nil {
		t.Fatal(err)
	}

	r.close()
	if _, err := e.AttachUSB(ctx, id, USBParams{DeviceID: d.ID, Fingerprint: d.Fingerprint}); err == nil {
		t.Fatal("uncertain owner was ignored")
	}

	r, err = e.openUSBRegistry(ctx)
	if err != nil {
		t.Fatal(err)
	}

	r.reconcile()
	if len(r.claims) != 1 || r.claims[0].Attachment.State != "unknown" {
		t.Fatal("uncertain claim lost")
	}

	r.close()
}

func TestUSBCaptureIdentityChangeCleansUp(t *testing.T) {
	e, id, d := usbTestEngine(t, t.TempDir())
	calls := 0
	e.usbDevices = func() ([]usb.Device, error) {
		calls++
		if calls > 1 {
			return []usb.Device{}, nil
		}

		return []usb.Device{d}, nil
	}
	if _, err := e.AttachUSB(context.Background(), id, USBParams{DeviceID: d.ID, Fingerprint: d.Fingerprint}); err == nil {
		t.Fatal("device changed during capture but operation succeeded")
	}

	attachments, err := e.ListVMUSB(context.Background(), id)
	if err != nil || len(attachments) != 0 {
		t.Fatalf("failed capture leaked assignment: %+v %v", attachments, err)
	}
}

func TestUSBAssignmentMatching(t *testing.T) {
	a := types.VMUSBAssignment{VendorID: 0x1050, ProductID: 0x0407, Serial: "S1"}
	one := usb.Device{VendorID: 0x1050, ProductID: 0x0407, Serial: "S1", State: "available"}
	two := usb.Device{VendorID: 0x1050, ProductID: 0x0407, Serial: "S2", State: "available"}
	if _, err := matchUSBAssignment(nil, a); err == nil {
		t.Fatal("missing device accepted")
	}

	if _, err := matchUSBAssignment([]usb.Device{one, two}, types.VMUSBAssignment{VendorID: 0x1050, ProductID: 0x0407}); err == nil {
		t.Fatal("ambiguous match accepted")
	}

	if _, err := matchUSBAssignment([]usb.Device{{VendorID: 0x1050, ProductID: 0x0407, Serial: "S1", State: "unsupported", Reason: "hub"}}, a); err == nil {
		t.Fatal("unavailable device accepted")
	}

	device, err := matchUSBAssignment([]usb.Device{one, two}, a)
	if err != nil || device.Serial != "S1" {
		t.Fatalf("serial match failed: %+v %v", device, err)
	}
}

func gateTestEngine(t *testing.T) *Engine {
	t.Helper()
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	e := New(paths)
	e.usbClaimsDir = filepath.Join(t.TempDir(), "usb")
	return e
}

func TestUSBBootGateDeniesMissingAndDoesNotStart(t *testing.T) {
	e := gateTestEngine(t)
	e.usbDevices = func() ([]usb.Device, error) { return []usb.Device{}, nil }
	m, err := e.CreateVM(CreateVMParams{Name: "gate", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	m.USB = []types.VMUSBAssignment{{VendorID: 0x1050, ProductID: 0x0407, Serial: "S1", Product: "Key"}}
	if err := e.vms.Save(m); err != nil {
		t.Fatal(err)
	}

	_, err = e.StartVM(context.Background(), m.ID)
	if err == nil || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("expected boot denied for missing device, got %v", err)
	}

	if st := e.driver.Status(m.ID); st.Phase == "running" {
		t.Fatal("VM started despite missing assigned device")
	}
}

func TestUSBBootGateDeniesClaimedDevice(t *testing.T) {
	e, _, d := usbTestEngine(t, t.TempDir())
	ctx := context.Background()
	if _, err := e.AttachUSB(ctx, "usb-test", USBParams{DeviceID: d.ID, Fingerprint: d.Fingerprint}); err != nil {
		t.Fatal(err)
	}

	other, err := e.CreateVM(CreateVMParams{Name: "gate-other", Image: "ubuntu-24.04-arm64", CPUs: 1, MemoryMiB: 128, DiskSizeGiB: 4}, nil)
	if err != nil {
		t.Fatal(err)
	}

	other.USB = []types.VMUSBAssignment{{VendorID: d.VendorID, ProductID: d.ProductID, Serial: d.Serial}}
	if err := e.vms.Save(other); err != nil {
		t.Fatal(err)
	}

	err = e.checkUSBAssignments(ctx, other)
	if err == nil || !strings.Contains(err.Error(), "already attached") {
		t.Fatalf("expected claimed-device denial, got %v", err)
	}
}

func TestUSBAssignUnassign(t *testing.T) {
	e, _, d := usbTestEngine(t, t.TempDir())
	d.Serial = "SER123"
	e.usbDevices = func() ([]usb.Device, error) { return []usb.Device{d}, nil }
	ctx := context.Background()
	a, err := e.AssignUSB(ctx, "usb-test", d.ID)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.AssignUSB(ctx, "usb-test", d.ID); err != nil {
		t.Fatal(err)
	}

	assignments, err := e.ListUSBAssignments("usb-test")
	if err != nil || len(assignments) != 1 {
		t.Fatalf("assign not idempotent: %+v %v", assignments, err)
	}

	if err := e.UnassignUSB(ctx, "usb-test", "dead:beef"); err == nil {
		t.Fatal("removed a nonexistent assignment")
	}

	if err := e.UnassignUSB(ctx, "usb-test", AssignmentKey(a)); err != nil {
		t.Fatal(err)
	}

	assignments, err = e.ListUSBAssignments("usb-test")
	if err != nil || len(assignments) != 0 {
		t.Fatalf("unassign failed: %+v %v", assignments, err)
	}
}

func TestStoppedUSBSelectionPersistsAndConnectsOnStart(t *testing.T) {
	e, id, device := usbTestEngine(t, t.TempDir())
	ctx := context.Background()
	pid := filepath.Join(e.paths.VMRunDir(id), "qemu.pid")
	if err := os.Remove(pid); err != nil {
		t.Fatal(err)
	}

	params := USBParams{DeviceID: device.ID, Fingerprint: device.Fingerprint}
	assignment, err := e.AssignUSBSelection(ctx, id, params)
	if err != nil {
		t.Fatal(err)
	}

	restored := New(e.paths)
	restored.usbClaimsDir = e.usbClaimsDir
	restored.usbDevices = e.usbDevices
	listed, err := restored.ListVMUSB(ctx, id)
	if err != nil || len(listed) != 1 || listed[0].State != "on-start" || listed[0].AssignmentKey != AssignmentKey(assignment) {
		t.Fatalf("saved assignment missing: %+v %v", listed, err)
	}

	if err := os.WriteFile(pid, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := restored.vms.Load(id)
	if err != nil {
		t.Fatal(err)
	}

	if err := restored.checkUSBAssignments(ctx, manifest); err != nil {
		t.Fatal(err)
	}

	if err := restored.attachAssignedUSB(ctx, manifest); err != nil {
		t.Fatal(err)
	}

	listed, err = restored.ListVMUSB(ctx, id)
	if err != nil || len(listed) != 1 || listed[0].State != "attached" || listed[0].AssignmentKey == "" {
		t.Fatalf("startup assignment not merged with live state: %+v %v", listed, err)
	}

	if err := restored.UnassignUSB(ctx, id, AssignmentKey(assignment)); err != nil {
		t.Fatal(err)
	}

	listed, err = restored.ListVMUSB(ctx, id)
	if err != nil || len(listed) != 0 {
		t.Fatalf("remove left assignment or device: %+v %v", listed, err)
	}
}

func TestStoppedUSBSelectionRejectsStaleFingerprint(t *testing.T) {
	e, id, d := usbTestEngine(t, t.TempDir())
	if err := os.Remove(filepath.Join(e.paths.VMRunDir(id), "qemu.pid")); err != nil {
		t.Fatal(err)
	}

	_, err := e.AssignUSBSelection(context.Background(), id, USBParams{DeviceID: d.ID, Fingerprint: strings.Repeat("f", 64)})
	if err == nil {
		t.Fatal("stale saved selection accepted")
	}

	assignments, err := e.ListUSBAssignments(id)
	if err != nil || len(assignments) != 0 {
		t.Fatalf("failed selection saved: %+v %v", assignments, err)
	}
}
