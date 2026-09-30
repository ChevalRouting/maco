# Architecture

maco turns a set of declarative YAML manifests into running virtual machines.
Whether a change comes from the CLI, the HTTP API, or the web UI, it flows
through the same pipeline.

## The pipeline

```
vms/*.yml -> load + validate -> desired state
                                     |
running QEMU processes -> actual state
                                     |
                          reconcile -> start / restart / destroy
                                     |
                          state cache (SQLite)
```

1. **Manifests** (`pkg/manifest`, `pkg/types`). Each VM is one YAML file at
   `<data-dir>/vms/<uuid>.yml`, parsed into a `types.VMManifest` and validated.
   The manifest set is the desired state and the source of truth.
2. **Driver** (`pkg/vm`). The driver turns a `vm.Spec` into a running
   `qemu-system-aarch64` process accelerated with `-accel hvf`. It manages the
   process lifecycle over QMP and a pid file: start, graceful powerdown, and
   liveness-based status.
3. **Images, firmware, and seed** (`pkg/image`, `pkg/vm/firmware`, `pkg/provision/cloudinit`).
   Base cloud images are downloaded and cached into the data directory. The UEFI
   firmware is a maco-branded ARM64 edk2 build (with the maco boot logo) embedded
   in the binary and extracted into the data directory on first use, so maco does
   not depend on host firmware. Per-VM qcow2 overlays are derived from the base
   images, and a NoCloud cloud-init seed ISO is built with `hdiutil` for
   first-boot config.
4. **Reconcile** (`maco reconcile`). maco boots every VM whose manifest sets
   `autostart` and is not already running, then refreshes the state cache from
   live process liveness. A full always-on daemon loop is still future work. Web mutations are executed by the Asynq worker started by `maco serve`.
5. **State cache** (`pkg/db`). A SQLite database, accessed through sqlc-generated
   queries and migrated with goose (`modernc.org/sqlite`, WAL), holds VM runtime
   state (phase, pid, boot time) in `vm_state`. It is a cache: deleting `web.db`
   and reconciling rebuilds it from the manifests and the running processes.

## Data layout

```
<data-dir>/
  vms/<uuid>.yml        VM manifests (source of truth)
  networks/<uuid>.yml   network topology manifests (source of truth)
  images/               base cloud image cache
  firmware/             maco-branded UEFI firmware, extracted from the binary
  disks/<uuid>/         per-VM qcow2 overlay and cloud-init seed
  web.db                SQLite runtime-state cache (vm_state)

/tmp/maco-<hash>/<uuid>/
  qemu.pid serial.log qmp.sock qemu.log
```

Runtime scratch defaults to `/tmp` rather than the data directory because macOS
caps unix-socket paths (used for QMP) at roughly 104 bytes, and the data
directory may be arbitrarily deep. A short per-data-dir hash keeps concurrent
maco instances from colliding. Set `MACO_WORKDIR` to relocate this scratch
elsewhere; keep the path short so socket paths stay under the limit.

## Privileges

maco runs as root. Host networking (native bridges, feth pairs, VLAN
interfaces) and vmnet modes require it, so maco calls `ifconfig` and its
privileged network worker directly instead of shelling out to `sudo`. The BPF
helper still drops privilege after opening its descriptor. QEMU only needs the
Hypervisor.framework entitlement (Homebrew signs `qemu-system-aarch64` with
`com.apple.security.hypervisor`) and user-mode NAT needs no privilege, but since
maco runs as root the VMs it launches run as root too. The data directory
defaults to `/Library/Application Support/maco`; override with `--data-dir` or
`MACO_DATA_DIR`.

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/maco` | the `maco` CLI (cobra commands, zerolog setup) |
| `pkg/config` | data-directory resolution and on-disk paths |
| `pkg/types` | shared types, including `VMManifest` |
| `pkg/manifest` | load, validate, and store VM manifests |
| `pkg/db` | SQLite runtime-state cache and users (sqlc queries, goose migrations) |
| `pkg/engine` | shared VM + network orchestration used by the CLI and the API |
| `pkg/jobs` | Asynq workers, Redis job history, and operation logs |
| `pkg/auth` | bcrypt password hashing and JWT issue/verify |
| `pkg/api` | chi HTTP API + embedded web UI (`maco serve`) |
| `web` | React + Vite + cheval-ui frontend, built into `pkg/api/dist` |
| `pkg/vm` | the QEMU/HVF process driver, QMP client, and interactive console |
| `pkg/net/vnet` | network topology manifests and their reconcile |
| `pkg/net/datapath` | native BPF/feth port supervisor, helper and cleanup lifecycle |
| `pkg/net/host` | the macOS host-networking layer (reads via stdlib `net`, mutates via `ifconfig`) |
| `pkg/image` | cloud-image download, cache, and overlay creation |
| `pkg/vm/firmware` | maco-branded UEFI firmware, embedded and extracted on use |
| `pkg/provision/cloudinit` | NoCloud seed ISO generation |

## Networking

Networks follow the same declarative model as VMs, applying a reconcile loop
to host interfaces on macOS. A network manifest describes a desired
host bridge; `maco networks apply` compares it against the live interfaces and
converges by creating the bridge, attaching existing and native VLAN members,
and assigning its optional address. VM start applies the referenced network.

`pkg/net/host` reads interface state through the standard library `net` package
and `ifconfig`, and mutates it with privileged `ifconfig` calls. Manifests
record which resources and attachments maco owns for later cleanup. Host
reconciliation and VM lifecycle operations use file locks; network manifests
are replaced atomically.

Each native bridge VM connects QEMU's Ethernet Unix stream to a BPF helper on
one endpoint of a newly created feth pair. The peer joins the native bridge.
A privileged maco worker supervises the helper and removes the pair on
connection closure, startup failure, or stop. QEMU and the helper run as the
calling user after setup. Network bridges persist until explicit destruction.
See [Network modes](network-modes.md) for the supported network types and
their setup.

## Web UI and API

`maco serve` runs a chi HTTP server (`pkg/api`) that exposes the engine over a
JSON API under `/api` and serves the React single-page app. The SPA is built
from `web/` into `pkg/api/dist` and embedded with `//go:build prod`
(`//go:embed all:dist`); `--dev-dir` serves an unembedded build instead. Non-API
routes fall back to `index.html` for client-side routing.

Authentication uses the SQLite `users` table (bcrypt) and stateless JWTs signed
with a secret at `<data-dir>/jwt.secret`; there is no unix-account dependency.
`POST /api/login` returns a token that the browser sends as a bearer header (held
in `sessionStorage`), and middleware guards every other `/api` route. Accounts
carry a role: `admin` accounts have full access, `viewer` accounts are read-only.
Mutating methods and the interactive consoles require `admin`; the role is read
from the database on every request. See [authentication.md](authentication.md).

The frontend depends on the shared `cheval-ui` design system via
`file:../../cheval-ui` (it lives beside the maco checkout at `/Users/vm/cheval-ui`).
The CLI, API, and UI all call `pkg/engine`, so a change made through any surface
lands in the same manifests and state cache.

## Background jobs and serial console

Every VM and network mutation under `/api` returns HTTP `202` with a job ID.
The Asynq worker uses a Redis queue scoped to the data directory and executes
operations through `pkg/engine`. Reads, authentication, and interactive serial
I/O remain immediate. CLI commands continue to call the engine synchronously.

`GET /api/jobs` lists all active operations and the latest 200 jobs. `GET /api/jobs/{id}` returns
status, submitted/started/finished timestamps, result resource ID, error, and
saved logs. Job metadata and logs remain in Redis independently of Asynq task
retention. Use Redis AOF persistence and include Redis in backups; `web.db`
does not contain job history. The history index keeps its most recent entries
and trims only finished jobs, so a pending or running job is never dropped;
private preview-capture jobs stay out of the index and expire on their own.

The worker has one execution slot. A Redis lock also serializes operations
across every instance, so host mutations never overlap when multiple servers
run (see [multi-instance.md](multi-instance.md)). Automatic retries are disabled
for host mutations. If a worker is interrupted after it begins an operation, a
redelivery fails rather than repeating uncertain side effects. Inspect the
resource and submit a new operation explicitly. Asynq crash recovery can take
until the worker lease expires before the job is reported failed.

VM details use an xterm.js terminal. `/api/vms/{id}/console` upgrades to a
WebSocket and requires a JWT in the first message within five seconds. Tokens
are not put in URLs. Same-origin checks remain enabled. After authentication,
the server resolves the manifest, checks the VM is running, replays up to
64 KiB from `serial.log`, and proxies the guest serial socket bidirectionally.
Authentication messages are limited to 4 KiB; authenticated console input is
limited to 1 MiB per WebSocket message. Serial I/O is a live session,
not a queued lifecycle operation. The existing QEMU serial socket is intended
for one active client; disconnect before using the CLI console.

The browser graphical console uses noVNC. Vite excludes `@novnc/novnc` from
dependency prebundling through `optimizeDeps.exclude`; the graphical console
still imports it and includes it in the production build.

Graphical access uses `/api/vms/{id}/display` with the same first-message JWT
and same-origin checks. After connecting to `vnc.sock`, the server sends
`{"type":"ready"}` and proxies binary RFB traffic. QEMU enables `virtio-gpu-pci`,
USB keyboard and tablet input, and VNC on a private Unix socket alongside the
serial socket. No public VNC TCP listener is opened. Restart the backend with a
rebuilt binary and stop/start existing VMs to enable these launch options.
A guest desktop must be installed separately if a desktop is desired.

### Job state WebSocket

`GET /api/jobs/stream` upgrades to a same-origin WebSocket. Send
`{"token":"<JWT>"}` as the first message within five seconds. The server then
sends `snapshot` events (`jobs`, an array) and `update` events (`job`, a single
job). An empty snapshot may omit `jobs`. Job metadata is persisted and its
state event is published atomically in one Redis transaction. The stream does
not expose Asynq payloads or guest credentials.

Each connection subscribes before reading its initial snapshot. Changes are
pushed through a Redis Pub/Sub channel scoped to the data directory, so workers
and API instances share updates. A snapshot every 20 seconds restores state
if Pub/Sub delivery was interrupted and reconciles archived Asynq tasks.
Pub/Sub is not durable event history; persisted job records are authoritative.

The jobs panel and jobs page share one browser WebSocket, with reconnect delays
from one to fifteen seconds. Reconnection loads a fresh snapshot. Job details
use streamed metadata while retrieving saved logs through the REST endpoint.

### VM previews

The job service checks running VMs at startup and every 30 seconds. It submits
`vm.screenshot` through Asynq and Redis, with a per-VM scheduling lease and an
outstanding-job check to avoid accumulating pending captures. Worker shutdown
cancels the scheduling loop before closing Redis. Captures use QMP `screendump`
in PNG format, take the VM process lock, and atomically replace `preview.png`
in the private VM runtime directory. The last successful image remains when a
capture fails or the VM stops; deletion removes the saved preview.

`GET /api/vms/{id}/preview` requires the usual Bearer authentication. The table
fetches image blobs every 30 seconds, with no token in the image URL, and revokes
object URLs on replacement or unmount. Before the first successful capture it
shows a display placeholder. Existing VMs require a stop/start with graphical
support enabled to produce screenshots.

### VM hardware management

Preview capture jobs are private backend jobs. Internal job access retains their
states and logs, while public REST lists and WebSocket snapshots exclude them.
Private updates are not published, and public job details return 404. Public
history scans past private jobs to retain the latest 200 public entries.

`GET /api/jobs` returns `items`, `page`, `page_size`, `total`, and `total_pages`.
Pages start at 1; the default size is 25 and the maximum is 100. Query parameters
`state` and `search` filter before counting and paging. The backend orders running
jobs first, pending jobs next, then completed jobs, newest first within each
group. `focus` puts a matching job first for action feedback. Out-of-range pages
are clamped to the last page. The UI renders the returned rows and pagination
metadata; stream updates invalidate the current page.

VM details provides Overview, CPU & memory, and Disks tabs. Hardware forms submit
`PATCH /api/vms/{id}/hardware` and return a `vm.hardware` job. CPU and memory updates require a
stopped VM and apply supplied fields only. CPU count must be at least one and
memory at least 64 MiB. The Disks tab grows the QCOW2 boot disk and adds, grows, or removes data disks. Capacity can grow, with checks
against both configured capacity and the actual image size to prevent shrinking.
The guest partition and filesystem may need to be expanded afterward. CPU and
memory changes take effect on the next start.

Data disks persist in the VM manifest with UUIDs, names, and capacities. QEMU
attaches each through `scsi-hd` on a persistent VirtIO SCSI controller with a stable serial UUID, so guests can
identify disks independently of device ordering. Disk operations submit
`vm.disk.add`, `vm.disk.grow`, or `vm.disk.remove` jobs and hold the VM
lifecycle lock. Running VMs attach disks through QMP `blockdev-add` and
`device_add`; growth uses `block_resize` without opening the active image with
qemu-img. VMs started before the SCSI controller was added need one restart
before live attachment works. Removal requires a stopped VM, uses a confirmation
dialog, and deletes the disk data; the boot disk cannot be removed. The Disks
tab displays square cards with actions at the bottom. Guest formatting and
filesystem expansion remain guest operations.

## USB passthrough

Host USB devices are enumerated with libusb. Authenticated inventory reads feed the VM USB picker; attach/detach jobs call the engine and QMP driver. The driver verifies capture and completed removal. A shared private runtime registry tracks exclusive assignments across data directories, while physical USB transfers go directly through QEMU. See [USB devices](usb.md) for connection identity, lifecycle, build requirements and verified limits.
