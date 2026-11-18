# Architecture

Every change, from the CLI, the API or the web UI, goes through the same engine
(`pkg/engine`) and lands in the same manifests and state cache.

```
vms/*.yml -> load + validate -> desired state
                                     |
running QEMU processes -> actual state
                                     |
                          reconcile -> start / restart / destroy
                                     |
                          state cache (SQLite)
```

1. **Manifests** (`pkg/manifest`, `pkg/types`): one YAML file per VM, parsed
   into a `types.VMManifest` and validated. This is the desired state.
2. **Driver** (`pkg/vm`): runs `qemu-system-aarch64 -accel hvf` from a
   `vm.Spec` and controls it over QMP and a pid file (start, graceful
   powerdown, status).
3. **Images, firmware, provisioning** (`pkg/image`, `pkg/vm/firmware`,
   `pkg/provision`): cloud images are downloaded into the data directory and
   each VM disk is a self-contained qcow2 copy of its image. The UEFI firmware
   is a maco-branded edk2 build embedded in the binary and extracted on first
   use. Guest setup is delivered as a cloud-init NoCloud seed ISO (built with
   `hdiutil`) or an Ignition config.
4. **Reconcile** (`maco reconcile`, and `maco serve` at startup): boots every
   VM with `autostart` that is not running, then refreshes the state cache.
5. **State cache** (`pkg/db`): VM phase, pid and boot time in `web.db`. Delete
   `web.db` and reconcile, and it is rebuilt from the manifests and running
   processes (users, API keys and job history are lost). See
   [database.md](database.md).

## Data layout

```
<data-dir>/
  vms/<uuid>.yml        VM manifests (source of truth)
  networks/<uuid>.yml   network manifests (source of truth)
  images/               downloaded cloud images
  templates/            VM templates
  disks/<uuid>/         VM disks and guest setup seed
  backups/<uuid>/       VM backups
  firmware/             UEFI firmware extracted from the binary
  web.db                SQLite database
  config.yaml           server settings (TLS)
  tls.crt, tls.key      self-signed certificate
  jwt.secret            session signing key

/var/run/maco-<hash>/<uuid>/
  qemu.pid  qemu.log  serial.log  qmp.sock  vnc.sock  preview.png
```

Runtime files live outside the data directory because macOS limits unix socket
paths to about 104 bytes. `<hash>` is derived from the data directory path, so
two data directories never collide. Running as root, maco uses `/var/run`
(cleared at boot). As another user it uses `/tmp`, where macOS deletes files
untouched for three days; if `qemu.pid` disappears, maco recovers the pid from
the process listening on the QMP socket. Set `MACO_WORKDIR` to move this
directory, and keep the path short.

## Privileges

maco runs as root: native bridges, feth pairs, VLAN interfaces and vmnet need
it, so maco calls `ifconfig` and its network helper directly instead of using
`sudo`. The helper drops privileges after opening its BPF descriptor. QEMU only
needs the `com.apple.security.hypervisor` entitlement, which the bundled
`qemu-system-aarch64` carries, but because maco runs as root, so do the VMs it
starts. See [security.md](security.md).

## Networking

A network manifest describes a host bridge. `maco networks apply` (and every
VM start) compares it with the live interfaces, then creates the bridge, joins
member and VLAN interfaces, and sets the optional address. The manifest records
what maco created, so destroying the network removes only that.

Each native bridge has one userspace switch, `maco-net-helper`, started with
the first VM that uses it. Every VM NIC is a port on that switch (a connected
socket handed to QEMU). The switch learns MAC addresses, forwards VM-to-VM
frames itself, and sends everything else through BPF on one end of a feth
pair whose peer is a member of the bridge. Host changes and VM lifecycle
operations take file locks. See [network-modes.md](network-modes.md).

## Web UI and API

`maco serve` runs a chi HTTP server (`pkg/api`) with the JSON API under `/api`
and the React app from `web/`. Production builds (`-tags prod`) embed the app;
`--dev-dir` serves a directory instead. Unknown non-API paths return
`index.html` so client-side routing works. The app uses the component library
in `ui/` (maco-ui). Authentication is covered in
[authentication.md](authentication.md), the API contract in [api.md](api.md).

## Jobs

Every VM and network mutation through the API returns `202` with a job. A
single worker runs jobs one at a time from a SQLite queue in `web.db`, so host
changes never overlap. A job interrupted by a crash or restart is marked failed,
never replayed: check the resource and submit it again. CLI commands call the
engine directly and do not go through the queue. Details in
[database.md](database.md); live updates in [notifications.md](notifications.md).

`GET /api/jobs` is paged (`page` from 1, `page_size` default 25, max 100) and
filtered by `state` and `search`. Running jobs come first, then pending, then
finished, newest first. `focus` puts one job at the top. `GET /api/jobs/{id}`
returns the job with its logs, and `GET /api/jobs/{id}/wait` streams until it
finishes.

## Consoles

The serial console (`/api/vms/{id}/console`) and the graphical console
(`/api/vms/{id}/display`) are WebSockets. The client sends the JWT as its first
message within five seconds; tokens never go in URLs, and the origin must
match. Both require the `admin` role.

- **Serial**: replays up to 64 KiB of `serial.log`, then proxies the guest
  serial socket. Input messages are limited to 1 MiB. Only one client should be
  attached at a time, so disconnect the browser before `maco vm console`.
- **Graphical**: QEMU exposes `virtio-gpu-pci`, a USB keyboard and tablet, and
  VNC on a private unix socket (no TCP listener). The browser uses noVNC. The
  guest needs its own desktop environment to show one.

## Previews

While `maco serve` runs, it captures a PNG of every running VM every 30 seconds
(QMP `screendump`) into `preview.png`. The last good image stays when a capture
fails or the VM stops. `GET /api/vms/{id}/preview` returns it. Capture jobs are
private: they never appear in job lists or streams.

## Hardware and disks

`PATCH /api/vms/{id}/hardware` changes CPUs (at least 1) and memory (at least
64 MiB) on a stopped VM; the change applies at the next start.

Data disks are stored in the manifest with a UUID, a name and a size, and are
attached as `scsi-hd` on a VirtIO SCSI controller with the UUID as serial, so
guests can identify them regardless of order. You can add or grow disks while
the VM runs; removing one needs a stopped VM and deletes its data. The boot
disk can grow but not be removed. Disks never shrink. After growing a disk,
expand the partition and filesystem inside the guest. A VM started by an older
maco needs one restart before live disk changes work.

## USB

libusb enumerates host devices and QEMU's `usb-host` device does the
passthrough. See [usb.md](usb.md).

## Metrics

While `maco serve` runs, it samples host CPU and memory, each QEMU process's
CPU and resident memory, and per-interface byte counters every five seconds.
Samples are kept in `web.db` for seven days. The Dashboard page charts them.

- VM memory is the QEMU process's resident memory (including overhead), not
  what the guest reports.
- VM CPU is a share of total host capacity.
- The first sample after a start or counter reset has no rate.
- Collection needs a cgo build.

`GET /api/metrics?scope=host&target=host&range=1h` returns a series. Use
`scope=vm&target=<vm-id>` or `scope=interface&target=<name>`, and list
interfaces with `GET /api/metrics/interfaces`. Ranges are `1h`, `6h`, `24h`
and `7d`.

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/maco` | the CLI (cobra) |
| `cmd/stylecheck` | the repository style checker |
| `pkg/api` | HTTP API and embedded web UI |
| `pkg/auth` | password hashing and JWTs |
| `pkg/config` | data directory and runtime paths |
| `pkg/db` | SQLite access (sqlc queries, goose migrations) |
| `pkg/engine` | VM, network, disk, backup and USB operations shared by CLI and API |
| `pkg/image` | cloud image catalog, download and VM disk creation |
| `pkg/jobs` | job queue (goqite), history and logs |
| `pkg/manifest`, `pkg/types` | manifest types, loading and validation |
| `pkg/metrics` | host and VM sampling |
| `pkg/net/host` | macOS interface reads and `ifconfig` changes |
| `pkg/net/vnet` | network manifests and their reconcile |
| `pkg/net/datapath` | native switch, feth pairs and cleanup |
| `pkg/provision` | cloud-init and Ignition seeds |
| `pkg/qemu` | locating and extracting the bundled QEMU |
| `pkg/service` | launchd daemon and log rotation |
| `pkg/tlscert` | certificates and ACME |
| `pkg/usb` | libusb device inventory |
| `pkg/vm` | QEMU driver, QMP client, consoles |
| `pkg/vm/firmware` | embedded UEFI firmware |
| `web` | React + Vite frontend |
| `ui` | maco-ui component library |
| `mcp` | the standalone MCP server (separate Go module) |
| `tools` | QEMU build, firmware build, packaging and deploy scripts |
