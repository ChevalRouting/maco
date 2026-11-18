# maco documentation

maco runs QEMU virtual machines on macOS (Apple Silicon), accelerated by
Hypervisor.framework. You describe each VM and network in a YAML manifest, and
maco makes the host match. The CLI, the HTTP API and the web UI all call the
same engine (`pkg/engine`), so anything one can do, the others can too.

- **Manifests** are the source of truth: one YAML file per VM in
  `<data-dir>/vms/` and one per network in `<data-dir>/networks/`.
- **`web.db`** is a SQLite database in the data directory. It caches runtime
  state and holds users, API keys, the job queue, job history and metrics.
- **`maco reconcile`** (also run by `maco serve` at startup) boots every VM with
  `autostart: true` and refreshes the cached state from the running QEMU
  processes.

maco runs as root, because it creates bridges, feth pairs and VLAN interfaces
on the host. The data directory defaults to `/Library/Application Support/maco`
(override with `--data-dir` or `MACO_DATA_DIR`).

To install it, see [install.md](install.md). To build and hack on it, see
[building.md](building.md).

## Pages

| Page | Covers |
|------|--------|
| [install.md](install.md) | install as a launchd daemon, the `.pkg`, deploy over SSH, logs, uninstall |
| [cli.md](cli.md) | `maco` command reference |
| [configuration.md](configuration.md) | VM and network manifests, autostart, TLS certificates |
| [network-modes.md](network-modes.md) | network modes and when to use each |
| [usb.md](usb.md) | USB passthrough: attach, assign, limits |
| [authentication.md](authentication.md) | accounts, roles, sessions, API keys |
| [security.md](security.md) | trust model and known limitations |
| [api.md](api.md) | the OpenAPI contract and client generation |
| [notifications.md](notifications.md) | the WebSocket event and job streams |
| [mcp.md](mcp.md) | the `maco-mcp` server for AI agents |
| [pwa.md](pwa.md) | installing the web UI on a Home Screen |
| [architecture.md](architecture.md) | how the pieces fit, data layout, repository layout |
| [database.md](database.md) | the SQLite database and the job queue |
| [building.md](building.md) | building maco, the patched QEMU, developing the UI |
| [testing.md](testing.md) | running the tests |
