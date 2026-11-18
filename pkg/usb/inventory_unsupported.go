//go:build !cgo

package usb

import "fmt"

func List() ([]Device, error) {
	return nil, fmt.Errorf("USB inventory requires macOS and a build with cgo and libusb 1.0.30 or newer")
}
