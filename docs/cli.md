# CLI reference

maco runs as root, so run these commands with `sudo` on an installed host.

Global flags:

| Flag | Default |
|------|---------|
| `--data-dir` | `$MACO_DATA_DIR`, or `/Library/Application Support/maco` |
| `--log-level`, `-v` | `info` (also `debug`, `warn`, `error`) |

`sudo` drops `MACO_DATA_DIR` by default, so use `--data-dir` with a custom data
directory.

Commands that take `REF` accept a VM or network UUID or name (UUID is tried
first).

## Images

```bash
maco image list                      # images in the catalog
maco image pull ubuntu-24.04-arm64   # download into <data-dir>/images/
```

## VMs

```bash
maco vm new NAME [flags]           # create a manifest
maco vm list                       # VMs with autostart and current state
maco vm start REF                  # boot under HVF
maco vm stop REF [--timeout 30s]   # graceful powerdown, then SIGTERM/SIGKILL
maco vm status REF                 # runtime status
maco vm console REF                # serial console, Ctrl+A then Q to detach
maco vm edit REF                   # edit the manifest in $EDITOR
maco vm rename REF NAME            # rename; the UUID, disks and backups stay
maco vm guest-agent REF            # guest OS details from qemu-guest-agent
maco vm destroy REF                # delete a stopped VM and its disks
maco reconcile                     # start autostart VMs, refresh cached state
```

`vm new` flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--image` | `ubuntu-24.04-arm64` | catalog image |
| `--cpus` | `2` | vCPUs |
| `--memory` | `2048` | memory in MiB |
| `--disk-size` | `20` | disk size in GiB |
| `--network` | (none) | network name or UUID; empty or `user` means user-mode NAT |
| `--address` | (none) | static guest IPv4/IPv6 CIDR, repeatable; DHCP otherwise |
| `--autostart` | `false` | start on `maco reconcile` and at daemon start |
| `--guest-config` | (none) | cloud-init user-data, or Butane for Flatcar, applied on first boot |

Without `--guest-config`, `vm new` turns guest setup off
(`guest_setup: {provisioner: none}`): no seed is attached, so no login user is
created and `--address` is not configured inside the guest. Cloud images have no
default password, so pass a `--guest-config` that creates a user (see
[configuration.md](configuration.md#guest-setup)), or create the VM from a
template in the web UI.

`vm edit` validates the manifest when you close the editor and saves it only if
it is valid; otherwise it asks whether to edit again or abort.

`vm console` detaches with `Ctrl+A` then `Q` and leaves the VM running. Guest
serial output is always written to `serial.log` in the VM's runtime directory.

On the first `vm start`, maco extracts its embedded UEFI firmware into
`<data-dir>/firmware/`.

## Backups and snapshots

```bash
maco vm backup REF                     # copy the disks into the backups folder (works while running)
maco vm backups REF                    # list backups
maco vm restore REF TIMESTAMP          # replace the VM with a backup
maco vm restore REF TIMESTAMP --as-new # restore as a new VM instead
maco vm backup-rm REF TIMESTAMP        # delete a backup
maco vm backup-prune REF               # delete backups the retention rules do not keep

maco vm snapshot create REF TAG [--ram] # --ram saves live memory (running VMs only)
maco vm snapshot list REF
maco vm snapshot restore REF TAG
maco vm snapshot rm REF TAG
```

Retention rules are set on the VM's backup schedule (Backups tab in the web UI,
or `setBackupSchedule`). They also run after every successful scheduled
backup. Backups are sorted newest first, then:

1. `keep_last` keeps the newest N.
2. `keep_daily`, `keep_weekly` (ISO weeks), `keep_monthly` and `keep_yearly`
   each keep the newest backup of N separate periods, in the host's local time,
   skipping periods an earlier rule already covers.
3. Everything else is deleted.

A rule set to 0 is off. With every rule at 0, nothing is deleted.

## Networks

```bash
maco networks new lab --address 192.0.2.1/24
maco networks apply lab --dry-run
maco networks apply lab
maco networks list
maco networks edit lab
maco networks destroy lab
```

`net` and `network` work as aliases of `networks`, and `create` of `new`.
`apply` without a name applies every network. Bridge networks are also applied
when a VM using them starts. `destroy` refuses a network that running VMs use,
or a bridge maco owns that has ports maco did not add.

`networks new` flags:

| Flag | Meaning |
|------|---------|
| `--mode` | `bridge` (default), `vmnet-bridged`, `user`, `switch` or `vlan` |
| `--address` | host IPv4 CIDR on the bridge |
| `--member` | existing interface to add to the bridge, repeatable |
| `--vlan` | VLAN member as `PARENT:TAG`, created by maco, repeatable |
| `--device` | use an existing bridge instead of creating one |
| `--uplink` | physical interface for `vmnet-bridged`, for example `en0` |
| `--parent`, `--tag` | parent interface and tag (1 to 4094) for `vlan` mode |

```bash
maco networks new access123 --vlan en0:123
maco networks new existing --device bridge3 --member vlan7
maco vm new web1 --network lab --address 192.0.2.2/24
```

A bridge has no DHCP or NAT of its own: give guests static addresses, or rely
on the LAN's services. Each bridge's switch logs to `bridges/<bridge>/network.log`
in the runtime directory, with port details in `network.json`. See
[network-modes.md](network-modes.md).

## USB

```bash
maco usb list [--json]                   # USB devices on the host
maco vm usb list VM                      # devices attached to a VM
maco vm usb attach VM DEVICE-ID          # attach until the VM stops
maco vm usb detach VM ATTACHMENT-ID      # release (eject in the guest first)
maco vm usb assign VM DEVICE-ID          # save in the manifest, attach on every start
maco vm usb assigned VM                  # list saved assignments
maco vm usb unassign VM VENDOR:PRODUCT[:SERIAL]
```

See [usb.md](usb.md).

## Accounts

```bash
maco user add NAME --password secret [--role admin|viewer]
maco user list
maco user passwd NAME --password new
maco user rm NAME
```

`--role` defaults to `admin`. See [authentication.md](authentication.md).

## Server

```bash
maco serve [--addr :8080]           # web UI and API
maco tls status                     # served certificates, issuer and expiry
maco version
```

`serve` flags:

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `:8080` | listen address |
| `--tls` | `true` | serve HTTPS; `--tls=false` for plain HTTP in development |
| `--tls-cert`, `--tls-key` | (none) | serve these files, overriding `config.yaml` |
| `--reconcile` | `true` | start autostart VMs and refresh state at startup |
| `--dev-dir` | (none) | serve the UI from a directory instead of the embedded one |

On first start, `serve` creates the `admin` account (password from
`MACO_ADMIN_PASSWORD`, or generated and logged once) and `<data-dir>/jwt.secret`.
Certificate settings are in [configuration.md](configuration.md#tls-certificates).
The HTTP API is described in [api.md](api.md).

## Service

```bash
sudo maco install                  # install binaries and QEMU, load the daemon
sudo maco service install          # write the launchd plists and load the daemon
sudo maco service status           # launchd state of the daemon and log rotation
sudo maco service rotate-logs      # rotate logs over 10 MiB now
sudo maco service uninstall        # unload the daemon and remove the plists
```

See [install.md](install.md).
