# Installing maco

maco installs as a launchd system daemon that runs `maco serve` (the web UI and
HTTPS API) as root and starts every VM with `autostart: true`.

A production build has no runtime dependencies. It embeds the web UI, the
network helper and a patched QEMU (`qemu-system-aarch64` with the
Hypervisor.framework entitlement, and `qemu-img`), and links libusb statically.
`maco install` extracts QEMU to `/usr/local/lib/maco/qemu`; maco uses that copy
even if Homebrew's `qemu` is installed.

## From a release

Releases are published at https://maco.chevalrouting.org with their
`SHA256SUMS`:

```bash
curl -LO https://maco.chevalrouting.org/downloads/maco-0.1.0-darwin-arm64.tar.gz
tar -xzf maco-0.1.0-darwin-arm64.tar.gz
sudo ./maco install
sudo maco user passwd admin --password 'choose-a-password'
```

The archive holds the production `maco` binary. The site also offers the same
build as a `.pkg` installer (see below).

## From a checkout

```bash
task build-ui
sudo ./maco install
sudo maco user passwd admin --password 'choose-a-password'
```

Then open https://localhost:8080 and sign in as `admin`. The certificate is
self-signed, so the browser warns once. To serve your own certificate or use
ACME, see [configuration.md](configuration.md#tls-certificates).

`maco install`:

1. copies `maco` to `/usr/local/bin` and writes `maco-net-helper` next to it,
2. extracts the bundled QEMU,
3. generates `tls.crt` and `tls.key` in the data directory if missing,
4. writes the launchd plists to `/Library/LaunchDaemons` and starts the daemon.

On first start the daemon creates an `admin` account with a generated password,
written to `/Library/Logs/maco/daemon.log`. Set your own with
`maco user passwd` as above. (`MACO_ADMIN_PASSWORD` only applies when you run
`maco serve` yourself.)

| Flag | Default | Meaning |
|------|---------|---------|
| `--addr` | `:8080` | listen address of the web UI and API |
| `--helper` | embedded | path to a `maco-net-helper` to install instead |
| `--tls-cert`, `--tls-key` | self-signed | serve these files (give both) |
| `--data-dir` | `/Library/Application Support/maco` | data directory for the daemon |

A plain `task build` (no `prod` tag) does not embed the helper or QEMU. Then
`maco install` uses the `maco-net-helper` next to the binary (or `--helper`),
and QEMU must be on `PATH`.

Upgrading from a version that used Redis removes its `com.maco.redis` daemon.
The Homebrew `redis` package is left alone; uninstall it if nothing else needs
it.

## With the .pkg installer

Building it needs Node for the web UI:

```bash
task pkg VERSION=0.1.0
sudo installer -pkg maco-0.1.0.pkg -target /
```

The package is unsigned: install it from the command line as above, or
right-click it in Finder and choose Open. Its `postinstall` runs
`maco install`.

## Deploy over SSH

From your development Mac:

```bash
task deploy HOST=admin@mac-mini.example
```

This makes a production build, uploads it to a new temporary directory on the
target, and runs there:

```bash
codesign --sign - --force --preserve-metadata=entitlements,requirements,flags,runtime maco
codesign --verify --strict maco
sudo ./maco service uninstall
sudo ./maco install
```

- Enter the remote sudo password when asked. `HOST` can be an SSH config alias
  (custom port, key, jump host).
- Signing happens before the old service is stopped. Uninstall waits up to 60
  seconds for the old daemon to exit, and stops the deploy if it does not.
- On success the staged files are removed. On failure they stay, their path is
  printed, and nothing is rolled back.
- Running VMs keep their QEMU until their next stop and start.
- A `maco serve` you started by hand is not touched.

The deploy does not read settings from the old plist. Pass install flags
after `--` if you use a custom data directory, address or certificate:

```bash
task deploy HOST=admin@mac-mini.example VERSION=0.1.0 -- \
  --data-dir '/Library/Application Support/maco' --addr :8443
```

## Managing the service

```bash
sudo maco service status      # launchd state of the daemon and log rotation
sudo maco service install     # re-register after changing flags
sudo maco service uninstall   # stop the daemon and remove the plists
```

`service install` takes `--addr`, `--tls-cert` and `--tls-key` and re-registers
the daemon without copying binaries or touching the certificate. Run it after
an upgrade, with your usual flags.

The daemon (`com.maco.daemon`) has `KeepAlive`: launchd starts it at boot and
restarts it if it exits, so `launchctl kill` will not keep it down. To control
it directly:

```bash
sudo launchctl kickstart -k system/com.maco.daemon   # restart
sudo launchctl bootout system/com.maco.daemon        # stop until next bootstrap or reboot
sudo launchctl bootstrap system /Library/LaunchDaemons/com.maco.daemon.plist  # start
```

### Logs

The daemon logs to `/Library/Logs/maco/daemon.log`. A second job,
`com.maco.logrotate`, checks it every hour and rotates it with macOS
`newsyslog` once it reaches 10 MiB, keeping seven gzipped archives
(`daemon.log.0.gz` and so on).

- Rotation briefly stops and restarts the daemon, so API connections can drop.
  A stopped daemon is not started.
- Only the daemon log is rotated, not VM logs.
- maco uses its own newsyslog config and adds nothing to `/etc/newsyslog.d`.

```bash
sudo maco service rotate-logs                     # rotate now
sudo launchctl print system/com.maco.logrotate
```

## Uninstalling

```bash
sudo maco service uninstall
sudo rm /usr/local/bin/maco /usr/local/bin/maco-net-helper
sudo rm -r /usr/local/lib/maco
```

This keeps logs, VM manifests, disks and the database. Delete
`/Library/Application Support/maco` to remove them too.
