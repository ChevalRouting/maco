# Testing

Unit tests live beside the packages they cover, in `pkg/` and `cmd/`. Run
everything with:

```bash
task test
```

That runs `go vet ./...` and `go test ./...`, plus a build check of the native
network helper, deployment script tests, and the independent MCP module's
vet and tests. `task test-mcp` runs only MCP checks; root `go test ./...`
does not traverse the nested `mcp/` module.

Run with coverage of the library packages:

```bash
go test ./... -coverpkg=./pkg/...
```

## Conventions

The suite is hermetic: no root, and no shared host state. Each test uses a
temporary data directory, and anything that would launch QEMU is exercised at
the argument-construction boundary or gated behind a build tag, so the default
run stays fast and side-effect free.

Representative coverage:

- Manifest save, load, list, resolve round-trips and validation.
- QEMU argument construction, QMP framing, and pid and status handling.
- Stable serial IDs for data disk attachment.
- `pkg/image` rejects an overlay smaller than its base image and accepts a
  sufficiently large one, using sparse QCOW2 fixtures and `qemu-img`.
- Sanitized C framing checks for the native helper: fragmented and coalesced
  stream input, 18-byte BPF capture headers, partial vectored writes, and
  per-frame fallback.

Privileged native networking and live guest boots are validated by hand rather
than in the default run, since they need root and mutate host interfaces.

`task test-deploy` exercises deployment scripts with mocked SSH, SCP, signing,
and sudo commands. Service unit tests cover delayed shutdown, bootout errors,
and cancellation before bootstrap. These checks do not modify launchd services
or connect to a remote host.
