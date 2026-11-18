# USB devices

You can pass a USB device plugged into the Mac through to a VM.

## In the web UI

Open the VM, then **USB devices > Add device**, and pick the device by name and
serial or port (**Device details** shows the technical data).

- **Stopped VM**: the device is saved in the manifest with an **On startup**
  badge and attached on every start. **Remove** deletes the assignment.
- **Running VM**: the device is attached now, until the VM stops. **Detach**
  releases it. Removing a saved assignment also detaches it.

## From the CLI

```sh
maco usb list [--json]
maco vm usb attach my-vm <device-id>       # until the VM stops
maco vm usb detach my-vm <attachment-id>
maco vm usb assign my-vm <device-id>       # saved, attached on every start
maco vm usb assigned my-vm                 # saved assignments and their keys
maco vm usb unassign my-vm <vendor:product[:serial]>
maco vm usb list my-vm
```

A device ID identifies the current physical connection, so unplugging and
replugging can change it. `assign` stores the vendor ID, product ID and serial,
never the bus address. `assigned` prints the key `unassign` expects.

## What to expect

- Attach and detach run as background jobs. A successful attach means QEMU has
  the device on the guest's USB bus. Whether it works then depends on the
  guest's drivers, so test serial, storage and security-key devices with your
  own hardware.
- Eject or unmount storage in the guest before detaching it. maco never
  unmounts filesystems on the host or in the guest.
- A composite device moves as a whole. The host loses access to it while it is
  attached.
- Host drivers or mounted storage can prevent capture, even as root.
- Hubs, and devices without a precise location, cannot be attached. Serial
  numbers may be missing when the host cannot open the device.

## Saved assignments at start

Saved assignments live under `usb:` in the manifest (vendor ID, product ID,
optional serial). Add a serial to tell two identical devices apart. Before
launching QEMU, maco matches each one against the connected devices:

- If a device is missing, matches more than one device, is unavailable, or is
  attached to another VM, the start fails and QEMU is not launched.
- If capturing a device fails after launch, maco stops the VM instead of
  running it without the device.

Keep saved devices plugged in when you start the VM.

## Unplugging

Unplugging a device does not detach it or stop the VM. The guest sees a normal
USB disconnect, and the UI shows **Disconnected** until you detach it. While it
stays attached, QEMU may reconnect any device that matches the same bus,
address, port and vendor/product IDs, even a different unit of the same model.
Detach before swapping devices. Refresh the USB tab to see changes.

## Ownership across restarts

maco records which VM owns which device in `/tmp/maco-usb-<uid>/claims.json`,
shared by every maco running as the same user, even with different data
directories. Restarting maco keeps devices attached to VMs that are still
running. If maco cannot inspect a VM's QEMU, the device stays reserved and the
UI shows **Check connection**. Do not delete this file while VMs with USB
devices are running.

## Requirements

- A cgo build with libusb 1.0.30 or newer. Production builds link libusb
  statically; a plain `task build` needs `brew install pkg-config libusb`, and
  libusb present at runtime.
- QEMU with `usb-host` (the bundled QEMU has it). The inventory API reports
  whether it is available.
- A VM created by an older maco needs one stop and start to get the USB
  controller.

## API

All routes need the usual bearer token.

| Method and path | Result |
|-----------------|--------|
| `GET /api/usb/devices` | `devices`, `supported`, and `reason` when unsupported |
| `GET /api/vms/{id}/usb` | saved and live assignments; saved ones include `assignment_key` |
| `POST /api/vms/{id}/usb` | attach now; body `device_id`, `fingerprint`; returns a `202` job |
| `DELETE /api/vms/{id}/usb/{attachment-id}` | detach; returns a `202` job |
| `POST /api/vms/{id}/usb/assignments` | save an assignment, also on a stopped VM; body `device_id`, `fingerprint` |
| `DELETE /api/vms/{id}/usb/assignments/{key}` | remove a saved assignment and detach it |

The jobs are `vm.usb.attach`, `vm.usb.detach`, `vm.usb.assign` and
`vm.usb.unassign`. Invalid requests fail before a job is queued. The
`fingerprint` is checked again just before and after capture, so a device that
changed in between fails the job. The CLI fills in the fingerprint for you.
