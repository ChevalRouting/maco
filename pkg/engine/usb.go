package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/storage"
	"github.com/m-vinc/maco/pkg/types"
	"github.com/m-vinc/maco/pkg/usb"
	"github.com/m-vinc/maco/pkg/vm"
)

type USBParams struct {
	DeviceID      string `json:"device_id"`
	Fingerprint   string `json:"fingerprint"`
	AttachmentID  string `json:"attachment_id,omitempty" binding:"optional"`
	AssignmentKey string `json:"assignment_key,omitempty" binding:"optional"`
}

func (p USBParams) ValidateAttach() error {
	if len(p.DeviceID) != 24 || len(p.Fingerprint) != 64 || p.AttachmentID != "" || p.AssignmentKey != "" {
		return fmt.Errorf("select a USB device from the current inventory")
	}

	return nil
}

type USBInventory struct {
	Devices   []usb.Device `json:"devices"`
	Supported bool         `json:"supported"`
	Reason    string       `json:"reason,omitempty" binding:"optional"`
}

type USBAttachment struct {
	AssignmentKey string     `json:"assignment_key,omitempty" binding:"optional"`
	ID            string     `json:"id"`
	VMID          string     `json:"vm_id"`
	VMName        string     `json:"vm_name"`
	Device        usb.Device `json:"device"`
	State         string     `json:"state"`
	Reason        string     `json:"reason,omitempty" binding:"optional"`
}

type usbClaim struct {
	Attachment USBAttachment `json:"attachment"`
	QEMUID     string        `json:"qemu_id"`
	RunDir     string        `json:"run_dir"`
	PID        int           `json:"pid"`
	Generation string        `json:"generation"`
}

type usbRegistry struct {
	dir    string
	lock   *os.File
	claims []usbClaim
}

func (r *usbRegistry) close() { _ = r.lock.Close() }

func (e *Engine) openUSBRegistry(ctx context.Context) (*usbRegistry, error) {
	dir := e.usbClaimsDir
	if dir == "" {
		dir = filepath.Join("/tmp", fmt.Sprintf("maco-usb-%d", os.Geteuid()))
	}

	file, err := storage.Lock(ctx, filepath.Join(dir, "lock"))
	if err != nil {
		return nil, err
	}

	r := &usbRegistry{dir: dir, lock: file, claims: []usbClaim{}}
	data, err := os.ReadFile(filepath.Join(dir, "claims.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		r.close()
		return nil, err
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &r.claims); err != nil {
			r.close()
			return nil, fmt.Errorf("USB registry unreadable: %w", err)
		}
	}

	return r, nil
}

func (r *usbRegistry) save() error {
	data, err := json.Marshal(r.claims)
	if err != nil {
		return err
	}

	return storage.WriteFile(filepath.Join(r.dir, "claims.json"), data, 0o600)
}

func usbGeneration(runDir string) (string, error) {
	info, err := os.Stat(filepath.Join(runDir, "qemu.pid"))
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(runDir, "qemu.pid"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)) + ":" + strconv.FormatInt(info.ModTime().UnixNano(), 10), nil
}

func (r *usbRegistry) reconcile() {
	retained := make([]usbClaim, 0, len(r.claims))
	for _, claim := range r.claims {
		generation, err := usbGeneration(claim.RunDir)
		if errors.Is(err, os.ErrNotExist) || (err == nil && generation != claim.Generation) {
			continue
		}

		if err == nil && syscall.Kill(claim.PID, 0) == syscall.ESRCH {
			continue
		}

		observed, observeErr := vm.ObserveUSB(filepath.Join(claim.RunDir, "qmp.sock"), claim.QEMUID)
		if err == nil && observeErr == nil && !observed.Exists {
			continue
		}

		claim.Attachment.State, claim.Attachment.Reason = "unknown", "Could not verify QEMU attachment; ownership is retained"
		if err == nil && observeErr == nil {
			claim.Attachment.State, claim.Attachment.Reason = "attached", ""
			if !observed.Attached {
				claim.Attachment.State, claim.Attachment.Reason = "missing", "Host device is disconnected or unavailable; detach to release the assignment"
			}
		}

		retained = append(retained, claim)
	}

	r.claims = retained
}

func (e *Engine) hostUSBDevices() ([]usb.Device, error) {
	if e.usbDevices != nil {
		return e.usbDevices()
	}

	return usb.List()
}

func (e *Engine) ListUSBDevices(ctx context.Context) (USBInventory, error) {
	result := USBInventory{Devices: []usb.Device{}}
	devices, err := e.hostUSBDevices()
	if err != nil {
		result.Reason = err.Error()
		return result, nil
	}

	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return result, err
	}

	defer r.close()
	r.reconcile()
	if err := r.save(); err != nil {
		return result, err
	}

	for i := range devices {
		if devices[i].State == "available" {
			if _, err := usb.Resolve(devices, devices[i].ID, devices[i].Fingerprint); err != nil {
				devices[i].State, devices[i].Reason = "unsupported", err.Error()
			}
		}

		for _, claim := range r.claims {
			if sameUSBPort(devices[i], claim.Attachment.Device) {
				devices[i].State = "assigned"
				devices[i].VMID, devices[i].VMName = claim.Attachment.VMID, claim.Attachment.VMName
				devices[i].Reason = "Assigned to " + claim.Attachment.VMName
			}
		}
	}

	result.Devices, result.Supported = devices, true
	if err := vm.USBSupport(); err != nil {
		result.Supported, result.Reason = false, err.Error()
	}

	return result, nil
}

func sameUSBPort(a, b usb.Device) bool { return a.Bus == b.Bus && a.Port == b.Port }

func (e *Engine) ListVMUSB(ctx context.Context, ref string) ([]USBAttachment, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return nil, err
	}

	defer r.close()
	r.reconcile()
	if err := r.save(); err != nil {
		return nil, err
	}

	result := []USBAttachment{}
	for _, claim := range r.claims {
		if claim.Attachment.VMID == m.ID && claim.RunDir == e.paths.VMRunDir(m.ID) {
			result = append(result, claim.Attachment)
		}
	}

	for _, assignment := range m.USB {
		key := AssignmentKey(assignment)
		found := false
		for i := range result {
			if assignmentMatchesDevice(assignment, result[i].Device) {
				result[i].AssignmentKey = key
				found = true
			}
		}

		if !found {
			product := assignment.Product
			if product == "" {
				product = fmt.Sprintf("USB device %04x:%04x", assignment.VendorID, assignment.ProductID)
			}

			result = append(result, USBAttachment{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte(m.ID+":"+key)).String(), VMID: m.ID, VMName: m.Name, AssignmentKey: key, State: "on-start", Device: usb.Device{Product: product, Serial: assignment.Serial, VendorID: assignment.VendorID, ProductID: assignment.ProductID, Classes: []string{}}})
		}
	}

	return result, nil
}

func assignmentMatchesDevice(a types.VMUSBAssignment, d usb.Device) bool {
	return a.VendorID == d.VendorID && a.ProductID == d.ProductID && (a.Serial == "" || a.Serial == d.Serial)
}

func (e *Engine) AttachUSB(ctx context.Context, ref string, p USBParams) (string, error) {
	if err := p.ValidateAttach(); err != nil {
		return "", err
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return "", err
	}

	return e.attachUSBDevice(ctx, m, func(devices []usb.Device) (usb.Device, error) {
		return usb.Resolve(devices, p.DeviceID, p.Fingerprint)
	})
}

func (e *Engine) attachUSBDevice(ctx context.Context, m *types.VMManifest, selector func([]usb.Device) (usb.Device, error)) (string, error) {
	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return "", err
	}

	defer func() { _ = lock.Close() }()
	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return "", err
	}

	defer r.close()
	st := e.driver.Status(m.ID)
	if st.Phase != vm.PhaseRunning {
		return "", fmt.Errorf("start the VM before attaching USB devices")
	}

	r.reconcile()
	devices, err := e.hostUSBDevices()
	if err != nil {
		return "", err
	}

	device, err := selector(devices)
	if err != nil {
		return "", err
	}

	for _, claim := range r.claims {
		if sameUSBPort(device, claim.Attachment.Device) {
			return "", fmt.Errorf("USB device is already assigned to %s", claim.Attachment.VMName)
		}
	}

	runDir := e.paths.VMRunDir(m.ID)
	generation, err := usbGeneration(runDir)
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	claim := usbClaim{Attachment: USBAttachment{ID: id, VMID: m.ID, VMName: m.Name, Device: device, State: "attaching"}, QEMUID: vm.USBDeviceID(id), RunDir: runDir, PID: st.PID, Generation: generation}
	r.claims = append(r.claims, claim)
	if err := r.save(); err != nil {
		return "", err
	}

	if err := e.driver.AttachUSB(m.ID, claim.QEMUID, device); err != nil {
		cleanupErr := e.driver.DetachUSB(m.ID, claim.QEMUID)
		if cleanupErr == nil {
			r.claims = r.claims[:len(r.claims)-1]
		} else {
			r.claims[len(r.claims)-1].Attachment.State = "unknown"
		}

		saveErr := r.save()
		return id, errors.Join(err, cleanupErr, saveErr)
	}

	fresh, verifyErr := e.hostUSBDevices()
	if verifyErr == nil {
		verifyErr = usb.VerifyConnection(fresh, device)
	}

	if verifyErr != nil {
		cleanupErr := e.driver.DetachUSB(m.ID, claim.QEMUID)
		if cleanupErr == nil {
			r.claims = r.claims[:len(r.claims)-1]
		}

		return id, errors.Join(fmt.Errorf("USB identity could not be confirmed after capture: %w", verifyErr), cleanupErr, r.save())
	}

	r.claims[len(r.claims)-1].Attachment.State = "attached"
	return id, r.save()
}

func assignmentLabel(a types.VMUSBAssignment) string {
	label := fmt.Sprintf("%04x:%04x", a.VendorID, a.ProductID)
	if a.Product != "" {
		label = a.Product + " (" + label + ")"
	}

	if a.Serial != "" {
		label += " serial " + a.Serial
	}

	return label
}

func AssignmentKey(a types.VMUSBAssignment) string {
	key := fmt.Sprintf("%04x:%04x", a.VendorID, a.ProductID)
	if a.Serial != "" {
		key += ":" + a.Serial
	}

	return key
}

func matchUSBAssignment(devices []usb.Device, a types.VMUSBAssignment) (usb.Device, error) {
	matches := []usb.Device{}
	for _, device := range devices {
		if device.VendorID != a.VendorID || device.ProductID != a.ProductID {
			continue
		}

		if a.Serial != "" && device.Serial != a.Serial {
			continue
		}

		matches = append(matches, device)
	}

	if len(matches) == 0 {
		return usb.Device{}, fmt.Errorf("assigned USB device %s is not connected", assignmentLabel(a))
	}

	if len(matches) > 1 {
		return usb.Device{}, fmt.Errorf("assigned USB device %s matches multiple connected devices; set a serial or connect only one", assignmentLabel(a))
	}

	if matches[0].State != "available" {
		return usb.Device{}, fmt.Errorf("assigned USB device %s is unavailable: %s", assignmentLabel(a), matches[0].Reason)
	}

	return matches[0], nil
}

func (e *Engine) checkUSBAssignments(ctx context.Context, m *types.VMManifest) error {
	if len(m.USB) == 0 {
		return nil
	}

	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return err
	}

	defer r.close()
	r.reconcile()
	if err := r.save(); err != nil {
		return err
	}

	devices, err := e.hostUSBDevices()
	if err != nil {
		return err
	}

	for _, assignment := range m.USB {
		device, err := matchUSBAssignment(devices, assignment)
		if err != nil {
			return err
		}

		for _, claim := range r.claims {
			if sameUSBPort(device, claim.Attachment.Device) {
				return fmt.Errorf("assigned USB device %s is already attached to %s", assignmentLabel(assignment), claim.Attachment.VMName)
			}
		}
	}

	return nil
}

func (e *Engine) attachAssignedUSB(ctx context.Context, m *types.VMManifest) error {
	for _, assignment := range m.USB {
		_, err := e.attachUSBDevice(ctx, m, func(devices []usb.Device) (usb.Device, error) {
			return matchUSBAssignment(devices, assignment)
		})
		if err != nil {
			return fmt.Errorf("attach assigned USB device %s: %w", assignmentLabel(assignment), err)
		}
	}

	return nil
}

func (e *Engine) AssignUSB(ctx context.Context, ref, deviceID string) (types.VMUSBAssignment, error) {
	return e.assignUSB(ctx, ref, deviceID, "")
}

func (e *Engine) AssignUSBSelection(ctx context.Context, ref string, p USBParams) (types.VMUSBAssignment, error) {
	if err := p.ValidateAttach(); err != nil {
		return types.VMUSBAssignment{}, err
	}

	return e.assignUSB(ctx, ref, p.DeviceID, p.Fingerprint)
}

func (e *Engine) assignUSB(ctx context.Context, ref, deviceID, fingerprint string) (types.VMUSBAssignment, error) {
	var empty types.VMUSBAssignment
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return empty, err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return empty, err
	}

	defer func() { _ = lock.Close() }()
	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return empty, err
	}

	defer r.close()
	m, err = e.vms.Load(m.ID)
	if err != nil {
		return empty, err
	}

	devices, err := e.hostUSBDevices()
	if err != nil {
		return empty, err
	}

	if fingerprint == "" {
		for _, device := range devices {
			if device.ID == deviceID {
				fingerprint = device.Fingerprint
			}
		}
	}

	device, err := usb.Resolve(devices, deviceID, fingerprint)
	if err != nil {
		return empty, err
	}

	assignment := types.VMUSBAssignment{VendorID: device.VendorID, ProductID: device.ProductID, Serial: device.Serial, Product: device.Product}
	if _, err := matchUSBAssignment(devices, assignment); err != nil {
		return empty, err
	}

	r.reconcile()
	for _, claim := range r.claims {
		if sameUSBPort(device, claim.Attachment.Device) && (claim.Attachment.VMID != m.ID || claim.RunDir != e.paths.VMRunDir(m.ID)) {
			return empty, fmt.Errorf("USB device is already assigned to %s", claim.Attachment.VMName)
		}
	}

	for _, existing := range m.USB {
		if AssignmentKey(existing) == AssignmentKey(assignment) {
			return assignment, nil
		}
	}

	m.USB = append(m.USB, assignment)
	if err := e.vms.Save(m); err != nil {
		return empty, err
	}

	return assignment, nil
}

func (e *Engine) UnassignUSB(ctx context.Context, ref, key string) error {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return err
	}

	defer r.close()
	m, err = e.vms.Load(m.ID)
	if err != nil {
		return err
	}

	matched := []types.VMUSBAssignment{}
	remaining := []types.VMUSBAssignment{}
	for _, assignment := range m.USB {
		if AssignmentKey(assignment) == key || fmt.Sprintf("%04x:%04x", assignment.VendorID, assignment.ProductID) == key {
			matched = append(matched, assignment)
		} else {
			remaining = append(remaining, assignment)
		}
	}

	if len(matched) == 0 {
		return fmt.Errorf("no USB assignment matches %q", key)
	}

	if len(matched) > 1 {
		return fmt.Errorf("ambiguous USB assignment %q; include its serial", key)
	}

	r.reconcile()
	retained := []usbClaim{}
	for _, claim := range r.claims {
		if claim.Attachment.VMID == m.ID && claim.RunDir == e.paths.VMRunDir(m.ID) && assignmentMatchesDevice(matched[0], claim.Attachment.Device) {
			if err := e.driver.DetachUSB(m.ID, claim.QEMUID); err != nil {
				return err
			}
		} else {
			retained = append(retained, claim)
		}
	}

	r.claims = retained
	if err := r.save(); err != nil {
		return err
	}

	m.USB = remaining
	return e.vms.Save(m)
}

func (e *Engine) ListUSBAssignments(ref string) ([]types.VMUSBAssignment, error) {
	m, err := e.vms.Resolve(ref)
	if err != nil {
		return nil, err
	}

	if m.USB == nil {
		return []types.VMUSBAssignment{}, nil
	}

	return m.USB, nil
}

func (e *Engine) DetachUSB(ctx context.Context, ref, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid USB attachment ID")
	}

	m, err := e.vms.Resolve(ref)
	if err != nil {
		return err
	}

	lock, err := e.driver.LockContext(ctx, m.ID)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Close() }()
	r, err := e.openUSBRegistry(ctx)
	if err != nil {
		return err
	}

	defer r.close()
	for i, claim := range r.claims {
		if claim.Attachment.ID != id {
			continue
		}

		if claim.Attachment.VMID != m.ID || claim.RunDir != e.paths.VMRunDir(m.ID) {
			return fmt.Errorf("USB attachment does not belong to this VM")
		}

		generation, generationErr := usbGeneration(claim.RunDir)
		if generationErr != nil && !errors.Is(generationErr, os.ErrNotExist) {
			return generationErr
		}

		if generationErr == nil && generation == claim.Generation && e.driver.Status(m.ID).Phase == vm.PhaseRunning {
			if err := e.driver.DetachUSB(m.ID, claim.QEMUID); err != nil {
				return err
			}
		}

		r.claims = append(r.claims[:i], r.claims[i+1:]...)
		return r.save()
	}

	return fmt.Errorf("USB attachment not found; refresh the device list")
}
