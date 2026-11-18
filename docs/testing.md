# Testing

```bash
task test
```

This runs:

- the native network helper tests, built with address and undefined-behavior
  sanitizers (fragmented and coalesced input, BPF headers, partial writes),
- the deployment script tests (`task test-deploy`), with SSH, SCP, signing and
  sudo mocked, so nothing touches launchd or a remote host,
- `go vet ./...` and `go test ./...`,
- vet and tests of the separate `mcp/` module (`task test-mcp`). A plain
  `go test ./...` at the root does not include it.

Coverage of the library packages:

```bash
go test ./... -coverpkg=./pkg/...
```

## Conventions

- Unit tests sit next to the code in `pkg/` and `cmd/`.
- Tests need no root and share no host state: each uses a temporary data
  directory.
- Code that would launch QEMU is tested at the argument-building level, or
  behind a build tag, so the default run is fast and has no side effects.
- Native networking with root and real guest boots are tested by hand, because
  they change host interfaces.
