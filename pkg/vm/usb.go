package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/m-vinc/maco/pkg/usb"
)

type USBObservation struct {
	Exists   bool `json:"exists"`
	Attached bool `json:"attached"`
}

type qomProperty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func USBDeviceID(id string) string { return "maco-usb-" + id }

func validUSBID(id string) bool {
	if !strings.HasPrefix(id, "maco-usb-") {
		return false
	}

	_, err := uuid.Parse(strings.TrimPrefix(id, "maco-usb-"))
	return err == nil
}

func openUSBQMP(socket string) (*qmpClient, error) {
	client, err := dialQMP(socket, 3*time.Second)
	if err != nil {
		return nil, err
	}

	if err := client.conn.SetDeadline(time.Now().Add(8 * time.Second)); err != nil {
		_ = client.close()
		return nil, err
	}

	return client, nil
}

func ObserveUSB(socket, id string) (USBObservation, error) {
	if !validUSBID(id) {
		return USBObservation{}, fmt.Errorf("invalid USB attachment ID")
	}

	client, err := openUSBQMP(socket)
	if err != nil {
		return USBObservation{}, err
	}

	defer func() { _ = client.close() }()
	return client.observeUSB(id)
}

func (c *qmpClient) observeUSB(id string) (USBObservation, error) {
	data, err := c.executeArguments("qom-list", map[string]string{"path": "/machine/peripheral"})
	if err != nil {
		return USBObservation{}, err
	}

	var properties []qomProperty
	if err := json.Unmarshal(data, &properties); err != nil {
		return USBObservation{}, err
	}

	for _, property := range properties {
		if property.Name != id {
			continue
		}

		if property.Type != "child<usb-host>" {
			return USBObservation{}, fmt.Errorf("USB attachment has an unexpected device type")
		}

		data, err = c.executeArguments("qom-get", map[string]string{"path": "/machine/peripheral/" + id, "property": "attached"})
		if err != nil {
			return USBObservation{}, err
		}

		var attached bool
		if err := json.Unmarshal(data, &attached); err != nil {
			return USBObservation{}, err
		}

		return USBObservation{Exists: true, Attached: attached}, nil
	}

	return USBObservation{}, nil
}

func (d *Driver) AttachUSB(vmID, id string, device usb.Device) error {
	if !validUSBID(id) {
		return fmt.Errorf("invalid USB attachment ID")
	}

	client, err := openUSBQMP(d.qmpPath(vmID))
	if err != nil {
		return err
	}

	defer func() { _ = client.close() }()
	args := map[string]any{"driver": "usb-host", "id": id, "bus": "maco-usb.0", "hostbus": device.Bus, "hostaddr": device.Address, "hostport": device.Port, "vendorid": device.VendorID, "productid": device.ProductID}
	if _, err := client.executeArguments("device_add", args); err != nil {
		return fmt.Errorf("attach USB (older VMs need a stop/start to enable the USB bus): %w", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		observed, err := client.observeUSB(id)
		if err != nil {
			return err
		}

		if observed.Attached {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("QEMU could not capture the USB device; check host access and unmount host storage before trying again")
}

func (d *Driver) DetachUSB(vmID, id string) error { return DetachUSBAt(d.qmpPath(vmID), id) }

func DetachUSBAt(socket, id string) error {
	if !validUSBID(id) {
		return fmt.Errorf("invalid USB attachment ID")
	}

	client, err := openUSBQMP(socket)
	if err != nil {
		return err
	}

	defer func() { _ = client.close() }()
	observed, err := client.observeUSB(id)
	if err != nil {
		return err
	}

	if !observed.Exists {
		return nil
	}

	if _, err := client.executeArguments("device_del", map[string]string{"id": id}); err != nil {
		return err
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, event := range client.events {
			if event.Event == "DEVICE_UNPLUG_GUEST_ERROR" {
				var data qmpDeviceEvent
				if json.Unmarshal(event.Data, &data) == nil && data.Device == id {
					return fmt.Errorf("guest rejected USB detach")
				}
			}
		}

		observed, err = client.observeUSB(id)
		if err != nil {
			return err
		}

		if !observed.Exists {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("USB detach is not confirmed; inspect the attachment before retrying")
}

type qmpDeviceEvent struct {
	Device string `json:"device"`
}

var usbSupportOnce sync.Once
var usbSupportError error

func USBSupport() error {
	usbSupportOnce.Do(probeUSBSupport)
	return usbSupportError
}

func probeUSBSupport() {
	binary, err := LocateQEMU()
	if err != nil {
		usbSupportError = err
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, "-device", "usb-host,help").CombinedOutput()
	if err != nil || !strings.Contains(string(output), "hostport") {
		usbSupportError = fmt.Errorf("installed QEMU does not expose USB host passthrough support")
	}
}
