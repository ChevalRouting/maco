# Building

maco builds and runs on macOS on Apple Silicon (`darwin/arm64`) only.

## Prerequisites

```bash
xcode-select --install                  # C toolchain for the network helper
brew install go-task pkg-config libusb
```

- **Go**: the version in `go.mod`. With the default `GOTOOLCHAIN=auto`, `go`
  downloads it for you.
- **Node and npm**: for the web UI (`task build-ui`, `task pkg`).
- **QEMU**: production builds embed maco's own (see below). A plain
  `task build` does not, and uses `qemu-system-aarch64` and `qemu-img` from
  `PATH`, so install Homebrew's `qemu` for development.

## Build

```bash
task build                 # ./maco and ./maco-net-helper, no web UI
task build-ui              # production build: UI, QEMU, helper embedded
```

| Task | What it does |
|------|--------------|
| `task build` | build the CLI and the network helper (`EMBED=1` for a production build) |
| `task build-ui` | production build with the embedded web UI, QEMU and helper |
| `task build-mcp` | build `maco-mcp` (see [mcp.md](mcp.md)) |
| `task pkg VERSION=x.y.z` | build an unsigned `.pkg` installer (see [install.md](install.md)) |
| `task deploy HOST=user@mac` | production build, then reinstall over SSH |
| `task publish HOST=user@host VERSION=x.y.z` | build the `.pkg` and upload it with `site/` and the screenshots to a static web host (`URL=` overrides https://maco.chevalrouting.org) |
| `task qemu` | build the patched QEMU into `build/qemu/qemu.tar.gz` |
| `task qemu-bench` | compare network throughput of Homebrew and patched QEMU |
| `task firmware-build` | rebuild the embedded UEFI firmware (needs Docker) |
| `task web-dev` | run the Vite dev server |
| `task test` | run all tests (see [testing.md](testing.md)) |
| `task check-style` | run every style, lint and type check (alias `task lint`) |
| `task api-generate`, `task api-check` | regenerate and verify the API contract (see [api.md](api.md)) |
| `task tidy` | `go mod tidy` in both Go modules |
| `task run -- vm list` | build, then run `./maco` with arguments |
| `task --list` | list every task |

`VERSION`, `GOFLAGS`, `GOTAGS`, `CC`, `BINARY` and `PKG` can be set on the
command line or in the environment.

A production build (`EMBED=1`, used by `build-ui`, `pkg` and `deploy`):

- builds with `-tags prod` and embeds the web UI, the network helper and the
  QEMU archive,
- links libusb statically, so the binary loads only system libraries (check
  with `otool -L maco`). libusb is LGPL-2.1 (source at
  https://github.com/libusb/libusb); you can relink maco against another
  libusb from this repository's source.

A plain `task build` embeds none of these and may link Homebrew's libusb
dynamically, so keep libusb installed to run it. Install `maco` and
`maco-net-helper` in the same directory.

## Patched QEMU

maco ships QEMU 11.1.1 with the patches in `tools/qemu/patches`. They remove
two per-packet costs on the network path (stack zeroing in virtio-net transmit
and one `writev` per frame in the stream backend), which raises VM-to-VM
throughput on a native bridge from about 6 to about 10 Gbit/s.

```bash
brew install dtc meson ninja pkg-config glib pixman libslirp libusb libpng jpeg-turbo zstd
task qemu
```

`task qemu` downloads the release named in `tools/qemu/VERSION`, checks it
against `tools/qemu/SHA256SUMS`, applies the patches and builds in
`build/qemu`. It rebuilds only when the patches, the script or the version
change.

The build keeps only what maco uses: `aarch64-softmmu` with HVF, slirp user
NAT, vmnet, libusb passthrough, VNC with PNG and JPEG, qcow2 with zstd, and
`qemu-img`. It copies the Homebrew libraries it links into `qemu/lib`, rewrites
their install names to `@rpath`, and signs everything ad hoc, keeping the
`com.apple.security.hypervisor` entitlement on `qemu-system-aarch64`. The
result (about 39 MB, 8.5 MB compressed) runs without Homebrew. `qemu/NOTICE`
and `share/doc/licenses` carry the GPLv2 source pointer and the bundled
libraries' licenses.

`maco install` extracts the embedded archive to `../lib/maco/qemu` next to the
binary (skipped when the checksum has not changed). maco looks for QEMU there
first, then on `PATH`.

The patches are plain `git format-patch` files and apply without maco:

```bash
curl -LO https://download.qemu.org/qemu-11.1.1.tar.xz
shasum -a 256 -c tools/qemu/SHA256SUMS
tar -xJf qemu-11.1.1.tar.xz
for patch in tools/qemu/patches/*.patch; do patch -d qemu-11.1.1 -p1 < "$patch"; done
```

### Benchmark

`task qemu-bench` boots two small Alpine guests with iperf3 (images checked
against `tools/qemu/bench/SHA256SUMS`) and measures TCP throughput between
them, one and four streams each way. It runs Homebrew's QEMU and the patched
build, first over a direct socket pair, then through `maco-net-helper`, with
the virtio-net settings maco uses on a native bridge. Pass
`STOCK=/path/to/qemu-system-aarch64` to compare another build. To run one
binary:

```bash
bash tools/qemu/bench/image.sh build/qemu/bench
python3 tools/qemu/bench/bench.py "$(which qemu-system-aarch64)" --image build/qemu/bench --mtu 9000
```

## Developing the web UI

Run the API and the Vite dev server side by side instead of rebuilding:

```bash
# terminal 1: API only (a plain build has no UI)
task build
MACO_ADMIN_PASSWORD=changeme ./maco serve --addr :8080 --tls=false

# terminal 2: Vite with hot reload
task web-dev
```

Open http://localhost:5173. Vite proxies `/api` to `:8080` (see
`web/vite.config.ts`). `--tls=false` is needed because `maco serve` defaults to
HTTPS with a self-signed certificate, which the proxy rejects.

The app imports the built `dist/` of `ui/`, and `task web-dev` builds it first.
After changing `ui/src`:

```bash
npm --prefix ui run build
npm --prefix ui run typecheck
npm --prefix ui run check:style
```

To check a production UI build without embedding it, run
`maco serve --dev-dir web/dist`.

## Linting

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
task check-style
```

`task check-style` runs the checker in `cmd/stylecheck` (Go rules and repository
text, which rejects em dashes), golangci-lint on both Go modules, and the
style, type and lint checks of `ui/` and `web/`.

## USB support

USB inventory needs cgo and libusb 1.0.30 or newer. A `CGO_ENABLED=0` build
still manages VMs but reports USB as unsupported. See [usb.md](usb.md).
