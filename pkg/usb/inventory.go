package usb

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type Device struct {
	ID           string   `json:"id"`
	Fingerprint  string   `json:"fingerprint"`
	Manufacturer string   `json:"manufacturer"`
	Product      string   `json:"product"`
	Serial       string   `json:"serial"`
	VendorID     uint16   `json:"vendor_id"`
	ProductID    uint16   `json:"product_id"`
	Bus          int      `json:"bus"`
	Address      int      `json:"address"`
	Port         string   `json:"port"`
	Session      string   `json:"session"`
	Classes      []string `json:"classes"`
	State        string   `json:"state"`
	Reason       string   `json:"reason,omitempty" binding:"optional"`
	VMID         string   `json:"vm_id,omitempty" binding:"optional"`
	VMName       string   `json:"vm_name,omitempty" binding:"optional"`
}

func identify(d *Device) {
	key := fmt.Sprintf("%d:%d:%s:%s:%04x:%04x", d.Bus, d.Address, d.Port, d.Session, d.VendorID, d.ProductID)
	d.ID = fmt.Sprintf("%x", sha256.Sum256([]byte(key)))[:24]
	d.Fingerprint = fmt.Sprintf("%x", sha256.Sum256([]byte(key+":"+d.Serial)))
	d.State = "available"
	if d.Product == "" {
		d.Product = "Unknown USB device"
	}

	if d.Address == 0 || d.Port == "" || (d.Session == "0" || d.Session == "") {
		d.State, d.Reason = "unsupported", "A precise host location and connection identity are unavailable"
	}

	for _, class := range d.Classes {
		if class == "Hub" {
			d.State, d.Reason = "unsupported", "USB hubs cannot be assigned"
		}
	}
}

func Resolve(devices []Device, id, fingerprint string) (Device, error) {
	for _, device := range devices {
		if device.ID != id {
			continue
		}

		if fingerprint == "" || device.Fingerprint != fingerprint {
			return Device{}, fmt.Errorf("USB selection changed; refresh and select the device again")
		}

		if device.State != "available" {
			return Device{}, fmt.Errorf("USB device unavailable: %s", device.Reason)
		}

		if err := uniqueSelector(devices, device); err != nil {
			return Device{}, err
		}

		return device, nil
	}

	return Device{}, fmt.Errorf("USB device disconnected or changed; refresh and select it again")
}

func className(code int) string {
	switch code {
	case 1:
		return "Audio"
	case 2, 10:
		return "Serial / communications"
	case 3:
		return "Human interface"
	case 6:
		return "Imaging"
	case 7:
		return "Printer"
	case 8:
		return "Storage"
	case 9:
		return "Hub"
	case 11:
		return "Smart card"
	case 14:
		return "Video"
	case 224:
		return "Wireless"
	case 239:
		return "Composite"
	case 255:
		return "Vendor specific"
	default:
		return fmt.Sprintf("Class %02x", code)
	}
}

func addClass(classes []string, code int) []string {
	if code == 0 {
		return classes
	}

	name := className(code)
	for _, existing := range classes {
		if existing == name {
			return classes
		}
	}

	return append(classes, name)
}

func cleanDescriptor(value string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}

		return r
	}, value))
}

func VerifyConnection(devices []Device, selected Device) error {
	if err := uniqueSelector(devices, selected); err != nil {
		return err
	}

	for _, device := range devices {
		if device.ID != selected.ID {
			continue
		}

		if device.Serial != "" && selected.Serial != "" && device.Serial != selected.Serial {
			return fmt.Errorf("USB serial number changed during capture")
		}

		return nil
	}

	return fmt.Errorf("USB connection changed during capture")
}

func uniqueSelector(devices []Device, selected Device) error {
	matches := 0
	for _, device := range devices {
		if selected.Bus != 0 && device.Bus != selected.Bus {
			continue
		}

		if device.Address != selected.Address || device.Port != selected.Port {
			continue
		}

		if selected.VendorID != 0 && device.VendorID != selected.VendorID {
			continue
		}

		if selected.ProductID != 0 && device.ProductID != selected.ProductID {
			continue
		}

		matches++
	}

	if matches != 1 {
		return fmt.Errorf("QEMU cannot uniquely identify this USB device; disconnect the matching device on another bus and refresh")
	}

	return nil
}
