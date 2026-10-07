# Building

maco ships a Go CLI and a small native networking helper for macOS on Apple Silicon (arm64). It drives QEMU
and Apple's Hypervisor.framework, so it is built and run on a Mac.

## Prerequisites

```bash
brew install qemu go-task
```

`qemu-system-aarch64` must carry the Hypervisor.framework entitlement, which
Homebrew's build provides:

```bash
codesign -d --entitlements - "$(which qemu-system-aarch64)" | grep hypervisor
# -> [Key] com.apple.security.hypervisor
```

cloud-init seed ISOs are built with the built-in `hdiutil`, so no extra tooling
is needed.

## Build

```bash
task build      # -> ./maco and ./maco-net-helper (darwin/arm64)
```

## Linux (KVM, x86_64)

maco also builds and runs on Linux with KVM on x86_64. The VM datapath uses a
tap device attached to a Linux bridge managed through netlink, so no native
helper is built.

Prerequisites: `qemu-system-x86_64`, OVMF firmware, access to `/dev/kvm` (add
the user to the `kvm` group), `iproute2`, and `xorriso` or `genisoimage` for
cloud-init seeds. Redis is required at runtime and is not managed by maco; run
it separately and point maco at it with `MACO_REDIS_URL`.

```bash
task build GOOS=linux GOARCH=amd64   # -> ./maco (single binary, no helper)
sudo ./maco install                  # binary, data dir, and systemd unit
```

`maco install` copies the binary to `/usr/local/bin`, creates the data dir at
`/var/lib/maco`, writes and enables a `maco.service` systemd unit, and verifies
that Redis is reachable. Set `MACO_OVMF` to override OVMF firmware discovery.

`task build` targets `darwin/arm64` by default. Run `task` for the default build or `task --list` for all tasks. Other useful tasks:

| Target | What it does |
|--------|--------------|
| `task build` | build the CLI and native networking helper |
| `task build-mcp` | build the standalone MCP server |
| `task build-ui` | build maco with the embedded web UI |
| `task deploy HOST=user@mac` | build production maco and reinstall it over SSH |
| `task test` | vet and test both Go modules, native helper and deployment scripts |
| `task test-mcp` | vet and test the independent MCP module |
| `task check-style` | build and run the Go coding-style checker |
| `task build-style-checker` | build `build/stylecheck` for the execution host |
| `task lint` | alias for `task check-style` |
| `task tidy` | tidy both Go modules |
| `task run -- vm list` | build then run with arguments |

The MCP lives in its own `mcp/` Go module and needs no local QEMU or Maco
daemon. See the [MCP walkthrough](mcp.md) for build and client configuration.

Task replaces the former Makefile; task names retain the previous target names.
For example, `task build EMBED=1` embeds the UI and `task pkg VERSION=0.1.0`
builds a versioned installer. `VERSION`, `GOOS`, `GOARCH`, `GOFLAGS`, `GOTAGS`,
`CC`, `BINARY`, and `PKG` can be overridden on the command line or through the
environment. `task run ARGS="vm list"` also remains supported.

## Toolchain

The module targets the Go version in `go.mod`. With `GOTOOLCHAIN=auto` (the
default), the `go` command downloads the matching toolchain automatically, so a
newer local Go is not required.

Install `maco` and `maco-net-helper` into the same directory. The helper is
compiled with the macOS C toolchain; install Command Line Tools with
`xcode-select --install` if needed. Host interface changes and native bridge VM ports require running maco as
root. Pass an explicit `--data-dir` when running under sudo. No setuid binary
or sudoers modification is installed.

## Redis for background jobs

`maco serve` requires Redis for its Asynq worker and saved job history. With
Docker and Docker Compose installed, start the included Redis service from
the repository root:

```bash
docker compose up -d --wait redis
```

`compose.yml` uses the official Redis image, publishes port 6379 only
on `127.0.0.1`, and stores data in the named `redis-data` volume. AOF persistence
is enabled with a one-second fsync interval. The `noeviction` policy keeps job
records from being evicted under memory pressure. Include the volume in backups.
Queued VM requests contain guest credentials, so protect Redis and its backups
as part of the maco data store.

Check or stop the service:

```bash
docker compose exec redis redis-cli ping
docker compose down
```

Stopping the service retains its volume and job history. `docker compose down
-v` deletes the volume and all saved Redis data.

The default URL is `redis://localhost:6379/0`. Override it with
`MACO_REDIS_URL` or `maco serve --redis-url redis://localhost:6379/1`. The HTTP
server does not start if Redis cannot be reached.

Install a compatible linter for the Go toolchain before running `task lint`:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
```

See [code-style.md](code-style.md#automated-validation) for UI lint dependencies,
pre-build checks and custom-rule tests.

## Developing the web UI

Do not rebuild and embed the SPA to iterate on it. Run the Go API and the Vite
dev server (HMR / React fast-refresh) as two processes:

```bash
# terminal 1: the API only (a plain build embeds no UI, so serve is API-only)
task build
MACO_REDIS_URL=redis://localhost:6379/0 MACO_ADMIN_PASSWORD=changeme ./maco serve --addr :8080 --tls=false

# terminal 2: the Vite dev server with hot reload
task web-dev
```

Open http://localhost:5173. Vite serves the app with hot module replacement and
proxies `/api` to `maco serve` on `:8080` (configured in `web/vite.config.ts`),
so there is no CORS setup and no rebuild between edits. `maco serve` defaults to
HTTPS with a self-signed certificate; pass `--tls=false` in dev so the Vite proxy
can reach it over plain HTTP.

Build the embedded, production UI only when you want a single binary that serves
it:

```bash
task build-ui        # npm run build -> pkg/api/dist, then go build -tags prod
./maco serve         # serves the embedded SPA (no Vite needed)
```

`maco serve --dev-dir web/dist` serves a one-off built directory without
embedding, which is mainly useful for checking a production build locally.

## USB inventory build dependency

The macOS build uses cgo and libusb 1.0.30 or newer for USB inventory. Install `pkg-config` and `libusb` alongside QEMU, and keep libusb installed at runtime. Builds with `CGO_ENABLED=0` retain VM management but report USB inventory as unsupported. See [USB devices](usb.md).
