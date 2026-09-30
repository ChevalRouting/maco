# CLI reference

All commands accept the global flags `--data-dir` (default `$MACO_DATA_DIR` or
`/Library/Application Support/maco`) and `--log-level` (`debug`, `info`, `warn`,
`error`).

## Images

```bash
maco image list                 # known images in the catalog
maco image pull ubuntu-24.04-arm64
```

`pull` downloads and caches a base cloud image under `<data-dir>/images/`.

## VMs

```bash
maco vm new NAME [flags]         # create a manifest (mints a uuid)
maco vm list                     # list VMs with autostart and cached state
maco vm start REF                # boot a VM under HVF (REF is a name or uuid)
maco vm stop REF [--timeout 30s] # graceful powerdown, then SIGTERM/SIGKILL
maco vm status REF               # runtime status of one VM
maco vm console REF              # attach to the serial console (Ctrl+A Q to quit)
maco vm edit REF                 # edit the manifest in $EDITOR, validated on save
maco vm guest-agent REF          # guest OS details from the qemu guest agent
maco vm backup REF               # snapshot the VM's disks into the backups folder
maco vm destroy REF              # delete a stopped VM and its disks
maco reconcile                   # autostart flagged VMs and refresh the cache
```

`reconcile` starts every VM whose manifest sets `autostart` and is not already
running, then writes each VM's observed phase and pid to the SQLite state cache
(`<data-dir>/web.db`). `vm list` refreshes the same cache from live status.

`console` attaches an interactive serial console to a running VM over its QMP-adjacent
socket. Type `Ctrl+A` then `Q` to detach; the VM keeps running. A full transcript is
always written to `serial.log` in the VM's runtime directory regardless of whether a
console is attached.

`vm new` flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--image` | `ubuntu-24.04-arm64` | base cloud image |
| `--cpus` | `2` | number of vCPUs |
| `--memory` | `2048` | memory in MiB |
| `--disk-size` | `20` | virtual disk size in GiB |
| `--user` | `maco` | primary login user |
| `--password` | `maco` | login password |
| `--ssh-key` | (none) | authorized SSH public key |
| `--network` | (empty) | named network or UUID; empty/`user` selects user-mode NAT |
| `--address` | (none) | static guest IPv4/IPv6 CIDR, repeatable; otherwise DHCPv4 |
| `--autostart` | `false` | boot this VM on `maco reconcile` |

`start`, `stop`, `status`, `console`, `edit`, `guest-agent`, `backup`, and
`destroy` resolve `REF` against the manifests by uuid first, then by name. `vm edit` opens the manifest in
`$EDITOR`, validates it on exit, and saves only if valid; an invalid manifest
prompts to re-edit or abort (visudo style).

On first `vm start`, maco extracts its embedded maco-branded UEFI firmware into
`<data-dir>/firmware/` and reuses it thereafter, so it does not depend on
firmware installed on the host. The firmware carries the maco boot logo, shown
during guest UEFI startup. Rebuild it with `task firmware-build` (Docker).

## Networks

```bash
maco networks create lab --address 10.100.0.1/24
maco networks apply lab --dry-run
maco networks apply lab
maco networks list
maco vm new web1 --network lab --address 10.100.0.2/24
maco vm start web1
maco vm stop web1
maco networks destroy lab
```

`net` and `network` are aliases for `networks`; `new` and `create` are aliases.
`apply [REF]` reconciles one network or all defined networks, and `edit REF`
opens its manifest in `$EDITOR`. Bridge networks are also applied automatically
when a VM starts. `destroy` rejects a network used by running VMs or an owned
bridge containing unmanaged ports.

```bash
maco networks create access123 --vlan en10:123
maco networks create access124 --vlan en10:124
maco networks create existing --device bridge3 --member vlan7
```

`--member` accepts existing interfaces, while repeatable `--vlan PARENT:TAG`
creates native VLAN members. maco runs as root and mutates host interfaces
directly (no sudo). QEMU uses a dedicated BPF/feth port on the native bridge;
the BPF helper drops privilege after opening its descriptor. Port interfaces are removed on
VM shutdown, QEMU disconnection, or startup failure; the network bridge remains
until explicitly destroyed. Logs and port details are in the VM runtime
directory as `network.log` and `network.json`.

An isolated bridge has no DHCP or NAT service. Use static guest addresses and
an optional host bridge address, or supply these services on the attached LAN.

`networks new` flags:

| Flag | Meaning |
|------|---------|
| `--mode` | `bridge`, `vmnet-bridged`, `user`, `switch`, or `vlan` (default `bridge`) |
| `--address` | optional host IPv4 CIDR on the bridge |
| `--member` | existing interfaces to join to the bridge, repeatable |
| `--vlan` | native VLAN members as `PARENT:TAG`, repeatable |
| `--device` | attach to an existing native bridge instead of creating one |
| `--uplink` | physical uplink interface for `vmnet-bridged`, e.g. `en0` |
| `--parent` / `--tag` | parent interface and `1..4094` tag for standalone `vlan` mode |

See [Network modes](network-modes.md) for what each mode does.

## USB

```bash
maco usb list                            # physical USB devices on the host
maco vm usb list VM                      # devices currently attached to a VM
maco vm usb attach VM DEVICE-ID          # live attach a host device to a VM
maco vm usb detach VM ATTACHMENT-ID      # release a device after ejecting it
maco vm usb assign VM DEVICE-ID          # persist a device so it reattaches on start
maco vm usb assigned VM                  # list persistent assignments in the manifest
maco vm usb unassign VM VENDOR:PRODUCT[:SERIAL]
```

Session attachments (`attach`) last until the VM stops; assignments (`assign`)
are written to the manifest and reattach on every start. See
[usb.md](usb.md) for discovery, matching, and limits.

## Accounts

Web UI accounts live in the SQLite database (bcrypt hashes), independent of unix
users.

```bash
maco user add admin --password secret [--role admin]
maco user add auditor --password secret --role viewer
maco user list
maco user passwd admin --password new
maco user rm bob
```

`--role` accepts `admin` (full access) or `viewer` (read-only). It defaults to
`admin`. See [authentication.md](authentication.md) for the permission model.

## Web UI and API

```bash
task build-ui                                    # build maco with the embedded SPA
MACO_ADMIN_PASSWORD=changeme maco serve --addr :8080
maco serve --dev-dir web/dist                    # serve an unembedded build (dev)
```

On first run `serve` creates an `admin` account (from `MACO_ADMIN_PASSWORD`, or a
generated password logged once) and a JWT secret at `<data-dir>/jwt.secret`.

## Service daemon

```bash
sudo maco service install     # write the launchd plists and start the daemon
sudo maco service status      # launchctl state for the maco and Redis daemons
sudo maco service uninstall   # unload the daemons and remove the plists
```

`service install` registers `maco serve` as a launchd system daemon
(`com.maco.daemon`) that also reconciles `autostart` VMs on boot, plus a
companion Redis daemon (`com.maco.redis`) when a `redis-server` binary is found.
It requires root. See [install.md](install.md) for the full setup and the `.pkg`
installer.

The HTTP API is JSON under `/api`, gated by a bearer JWT from `POST /api/login`:

| Method + path | Purpose |
|---------------|---------|
| `POST /api/login` | exchange username/password for a JWT |
| `GET/POST /api/vms`, `GET/DELETE /api/vms/{id}` | list, create, inspect, delete VMs |
| `POST /api/vms/{id}/start\|stop` | start or stop a VM |
| `GET/POST /api/networks`, `POST /api/networks/{id}/apply`, `DELETE /api/networks/{id}` | manage networks |
| `GET /api/images` | image catalog |

The CLI, API, and UI all drive the same `pkg/engine`, so they never diverge.
