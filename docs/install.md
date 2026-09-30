# Installing maco

maco installs as a macOS system daemon. A `.pkg` installer places the binaries
under `/usr/local/bin` and registers a launchd service that runs `maco serve`
(the web UI and HTTPS API) and starts every VM whose manifest sets `autostart`.
The daemon runs as root because maco manages host networking directly.

The web UI and API are always served over TLS. On install maco generates a
self-signed certificate (`tls.crt` and `tls.key` in the data directory) if one
is not already present, so the first connection is `https://localhost:8080`. The
browser warns about the self-signed certificate; accept it to continue. Drop a
CA-issued `tls.crt` and `tls.key` into the data directory to use your own.

## Prerequisites

maco needs QEMU with the Hypervisor.framework entitlement and a running Redis.
Both come from Homebrew:

```bash
brew install qemu redis
```

The `qemu` formula provides both `qemu-system-aarch64` (the hypervisor) and
`qemu-img` (used to build and expand VM disks); maco needs both. Confirm QEMU
carries the HVF entitlement:

```bash
codesign -d --entitlements - "$(which qemu-system-aarch64)" | grep hypervisor
```

The launchd daemon runs with a `PATH` that includes the Homebrew locations
(`/opt/homebrew/bin`, `/usr/local/bin`), so it finds `qemu-img` and
`qemu-system-aarch64` even though launchd does not inherit your shell `PATH`. If
QEMU is installed somewhere else, symlink the tools into one of those
directories.

Redis backs the background job queue and saved job history. maco does not bundle
it. The installer looks for `redis-server` on `PATH` and at the standard
Homebrew locations (`/opt/homebrew/bin/redis-server`,
`/usr/local/bin/redis-server`) and, when found, registers it as a companion
launchd daemon (`com.maco.redis`) bound to `127.0.0.1:6379`. Install Redis
before maco so the companion daemon is set up automatically; otherwise `maco
serve` retries until Redis becomes reachable.

## Install with the .pkg

Build the installer from a checkout (requires Node for the embedded UI):

```bash
task pkg VERSION=0.1.0
```

This produces `maco-0.1.0.pkg`. It is unsigned, so install it from the command
line or right-click and choose Open in Finder to bypass Gatekeeper:

```bash
sudo installer -pkg maco-0.1.0.pkg -target /
```

The installer's `postinstall` runs `maco install`, which installs the binaries,
generates the TLS certificate, writes the launchd plists to
`/Library/LaunchDaemons`, loads them, and starts the daemon. Open
https://localhost:8080 once it is running. On first start maco creates an
`admin` account with a generated password, logged to
`/Library/Logs/maco/daemon.log`. Set your own with `MACO_ADMIN_PASSWORD` before
the first run, or change it later with `maco user passwd admin`.

## Install from a local build

Without the installer, build and register everything in one step:

```bash
task build-ui
sudo ./maco install
```

`maco install` copies `maco` to `/usr/local/bin`, writes `maco-net-helper`
beside it, generates the self-signed TLS certificate, and loads the daemon. It
accepts:

- `--addr`: listen address for the web UI and API (default `:8080`).
- `--redis-server`: path to a `redis-server` binary to run as the companion
  daemon. Defaults to the first match on `PATH` or a Homebrew location.
- `--no-redis`: do not manage a Redis daemon; use an external Redis reachable
  at `MACO_REDIS_URL` (default `redis://localhost:6379/0`).
- `--helper`: path to a `maco-net-helper` binary to install instead of the
  embedded one.
- `--tls-cert` / `--tls-key`: serve with your own certificate and key instead
  of the generated self-signed pair. Both must be given together; the daemon is
  pointed at these paths.

`task build` compiles `maco-net-helper` and embeds it into the `maco` binary, so a
production build (`task build-ui` or the `.pkg`) is self-contained: `maco
install` extracts the helper from itself. A plain `go build` (no `prod` tag) has
no embedded helper; in that case `maco install` falls back to a `maco-net-helper`
sitting next to the binary, or pass `--helper`. `maco service install` remains
available to re-register the launchd daemon without recopying binaries or
regenerating the certificate.

## Deploy over SSH

From your development Mac:

```bash
task deploy HOST=admin@mac-mini.local
```

The task builds production Maco with the embedded UI and networking helper,
creates a temporary directory on the destination, and uploads the binary and
installer script with SCP. It then opens an SSH terminal and runs:

```bash
codesign --sign - --force --preserve-metadata=entitlements,requirements,flags,runtime maco
codesign --verify --strict maco
sudo ./maco service uninstall
sudo ./maco install
```

Enter the remote account's sudo password when prompted. `HOST` can also be an
SSH config alias, allowing custom ports, identity files, and jump hosts. The
destination must already have Maco's runtime dependencies installed. The build
defaults to Apple Silicon (`darwin/arm64`).

Pass installer options after `--`, particularly when the existing installation
uses a custom data directory, address, or TLS certificate:

```bash
task deploy HOST=admin@mac-mini.local VERSION=0.1.0 -- \
  --data-dir '/Library/Application Support/maco' --addr :8443
```

Without these options, `maco install` uses its normal defaults; deployment does
not recover settings from the old launchd plist. Uninstall leaves VM data and
certificates in place, but stops the Maco daemon and its managed Redis service.

Service uninstall and reload now report launchctl errors and wait up to 60
seconds per service for the launchd registration and old process to disappear.
If shutdown fails or times out, installation stops. This prevents bootstrapping
a replacement while the old service is still exiting. A manually started
`maco serve` process is not managed by this task.

Signing completes before services are stopped. Uploads use a fresh staging
directory instead of overwriting the running binary. Successful deployment
removes the staged files; a failed remote installation leaves them in place
and prints their location for diagnosis. Deployment does not automatically
roll back a failed installation.

## Managing the service

```bash
sudo maco service status      # launchctl state for both daemons
sudo maco service install     # re-install and reload after upgrading the binary
sudo maco service uninstall   # unload the daemons and remove the plists
```

Logs are written to `/Library/Logs/maco/daemon.log` and
`/Library/Logs/maco/redis.log`. The daemon uses `KeepAlive`, so launchd restarts
it if it exits and starts it again at boot.

To stop or start the running daemon directly with `launchctl` (without removing
the plists):

```bash
sudo launchctl kickstart -k system/com.maco.daemon   # restart
sudo launchctl bootout system/com.maco.daemon         # stop
sudo launchctl bootstrap system /Library/LaunchDaemons/com.maco.daemon.plist  # start
```

Because `KeepAlive` is set, `launchctl kill` alone will not keep the daemon down;
launchd relaunches it. Use `bootout` to stop it until the next `bootstrap` or
reboot, or `maco service uninstall` to remove it entirely. Replace
`com.maco.daemon` with `com.maco.redis` to control the companion Redis daemon.

## Uninstalling

```bash
sudo maco service uninstall
sudo rm /usr/local/bin/maco /usr/local/bin/maco-net-helper
```

VM manifests, disks, and the state cache under
`/Library/Application Support/maco` are left in place. Remove that directory to
discard them.
