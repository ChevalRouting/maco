//go:build darwin && cgo

package usb

/*
#cgo pkg-config: libusb-1.0
#include <libusb.h>
#include <stdlib.h>
#if LIBUSB_API_VERSION < 0x0100010C
#error "maco USB inventory requires libusb 1.0.30 or newer"
#endif

static int interface_class(struct libusb_config_descriptor *c, int i) {
 if (c->interface[i].num_altsetting < 1) return 0;
 return c->interface[i].altsetting[0].bInterfaceClass;
}
*/
import "C"

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

func List() ([]Device, error) {
	var ctx *C.libusb_context
	if result := C.libusb_init(&ctx); result != 0 {
		return nil, fmt.Errorf("USB inventory: %s", C.GoString(C.libusb_error_name(result)))
	}

	defer C.libusb_exit(ctx)
	var list **C.libusb_device
	count := C.libusb_get_device_list(ctx, &list)
	if count < 0 {
		return nil, fmt.Errorf("USB inventory failed: %d", count)
	}

	defer C.libusb_free_device_list(list, 1)
	devices := make([]Device, 0, int(count))
	for _, device := range unsafe.Slice(list, int(count)) {
		var descriptor C.struct_libusb_device_descriptor
		if C.libusb_get_device_descriptor(device, &descriptor) != 0 {
			continue
		}

		d := Device{VendorID: uint16(descriptor.idVendor), ProductID: uint16(descriptor.idProduct), Bus: int(C.libusb_get_bus_number(device)), Address: int(C.libusb_get_device_address(device)), Session: strconv.FormatUint(uint64(C.libusb_get_session_data(device)), 10), Classes: []string{}}
		var ports [7]C.uint8_t
		length := int(C.libusb_get_port_numbers(device, &ports[0], 7))
		path := []string{}
		for i := 0; i < length; i++ {
			path = append(path, strconv.Itoa(int(ports[i])))
		}

		d.Port = strings.Join(path, ".")
		d.Classes = addClass(d.Classes, int(descriptor.bDeviceClass))
		var config *C.struct_libusb_config_descriptor
		if C.libusb_get_config_descriptor(device, 0, &config) == 0 {
			for i := 0; i < int(config.bNumInterfaces); i++ {
				d.Classes = addClass(d.Classes, int(C.interface_class(config, C.int(i))))
			}

			C.libusb_free_config_descriptor(config)
		}

		var handle *C.libusb_device_handle
		if C.libusb_open(device, &handle) == 0 {
			d.Manufacturer = readDescriptor(handle, descriptor.iManufacturer)
			d.Product = readDescriptor(handle, descriptor.iProduct)
			d.Serial = readDescriptor(handle, descriptor.iSerialNumber)
			C.libusb_close(handle)
		}

		identify(&d)
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Bus != devices[j].Bus {
			return devices[i].Bus < devices[j].Bus
		}

		return devices[i].Port < devices[j].Port
	})
	return devices, nil
}

func readDescriptor(handle *C.libusb_device_handle, index C.uint8_t) string {
	if index == 0 {
		return ""
	}

	var buffer [256]C.uchar
	length := C.libusb_get_string_descriptor_ascii(handle, index, &buffer[0], 256)
	if length < 0 {
		return ""
	}

	return cleanDescriptor(C.GoStringN((*C.char)(unsafe.Pointer(&buffer[0])), length))
}
