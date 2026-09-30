# maco documentation

maco is a lightweight VMM for macOS. You describe the virtual machines you want
in YAML, and maco reconciles the running system to match: it starts, restarts,
and destroys QEMU virtual machines accelerated by Apple's Hypervisor.framework.

The YAML manifests are the source of truth. A SQLite database caches runtime
state only, so it can always be rebuilt from the manifests. A web UI and an HTTP
API sit on top of the same engine (`pkg/engine`), and everything they can do has
a CLI equivalent.

## Getting started

maco needs QEMU with the Hypervisor.framework entitlement, which Homebrew's
build provides:

```bash
brew install qemu pkg-config libusb
codesign -d --entitlements - "$(which qemu-system-aarch64)" | grep hypervisor
```

Build the binary and boot a VM:

```bash
task build

./maco image pull ubuntu-24.04-arm64
./maco vm new web1 --cpus 2 --memory 2048
./maco vm start web1
./maco vm list
./maco vm stop web1
```

Serve the web UI and HTTP API (start [Redis](building.md#redis-for-background-jobs)
and build with the embedded UI first):

```bash
task build-ui
MACO_REDIS_URL=redis://localhost:6379/0 MACO_ADMIN_PASSWORD=changeme ./maco serve --addr :8080
```

On first run maco creates an `admin` account (from `MACO_ADMIN_PASSWORD`, or a
generated password logged once). Manage accounts with `maco user add|list|passwd|rm`,
each with an `admin` or `viewer` role ([authentication.md](authentication.md)).
Log in at https://localhost:8080. The UI has no unix-account dependency; accounts
live in the SQLite database with bcrypt hashes and the browser holds a JWT.

maco runs as root: it manages host networking (native bridges, feth pairs, VLAN
interfaces) and privileged VM networking directly, without shelling out to
`sudo`. QEMU reaches HVF through its own signing entitlement. The data directory
defaults to `/Library/Application Support/maco` (override with `--data-dir` or
`MACO_DATA_DIR`).

## How it fits together

- **Manifests**. One YAML file per VM at `<data-dir>/vms/<uuid>.yml`, the source
  of truth for a VM's spec and desired state. See
  [configuration.md](configuration.md).
- **State cache**. A SQLite database holds runtime state (power state, pid) as a
  cache, always rebuildable from the manifests.
- **Reconcile**. `maco reconcile` boots every VM whose manifest sets
  `autostart` and is not already running, and refreshes the state cache from the
  actual QEMU processes.
- **Surfaces**. The `maco` CLI, the HTTP API, and the web UI are three views of
  the same engine (`pkg/engine`); the API and UI are gated by JWT login.

## Documentation index

### Overview

| Page | Purpose |
|------|---------|
| [architecture.md](architecture.md) | the manifest/reconcile/driver pipeline and repository layout |
| [usb.md](usb.md) | host USB discovery, VM attach/detach and current limits |
| [configuration.md](configuration.md) | the VM manifest model: fields, desired state, storage |

### Surfaces and operations

| Page | Purpose |
|------|---------|
| [install.md](install.md) | installing maco as a launchd daemon and building the .pkg |
| [pwa.md](pwa.md) | installing the online web app and deferred iPad validation |
| [mcp.md](mcp.md) | agent access through the standalone multi-instance MCP server |
| [cli.md](cli.md) | `maco` command reference |
| [authentication.md](authentication.md) | accounts, JWT sessions, and the admin/viewer role model |
| [security.md](security.md) | trust model, root/QEMU isolation limits, Redis auth, image integrity |
| [multi-instance.md](multi-instance.md) | running several servers against one Redis and data directory |

### Development

| Page | Purpose |
|------|---------|
| [building.md](building.md) | building the binary on macOS and the prerequisites |
| [testing.md](testing.md) | test layout and conventions |
| [code-style.md](code-style.md) | coding rules gofmt does not enforce |

### Internal

These are design and planning notes kept in the repository but excluded from the
published documentation site.

| Page | Purpose |
|------|---------|
| [internal/system-flow.md](internal/system-flow.md) | diagrams of UI, API, jobs, QEMU and filesystem flows |
| [internal/manifest-schema-plan.md](internal/manifest-schema-plan.md) | versioned manifest schemas, validation, and MCP discovery |
| [internal/native-apple-app-plan.md](internal/native-apple-app-plan.md) | native Mac/iPad app milestones, SwiftUI navigation, and wireframes |
| [internal/usb-passthrough-plan.md](internal/usb-passthrough-plan.md) | USB implementation plan and follow-up scope |

See [Network modes](network-modes.md) for native bridge/VLAN, vmnet physical uplink, and User NAT creation.

See [UI notifications](notifications.md) for the shared WebSocket protocol and resource refresh behavior.

[API contract and client generation](api.md) describes the served OpenAPI spec
and the regeneration checks.
