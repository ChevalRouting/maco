# maco-mcp

A stdio MCP server that manages one or more maco hosts through their HTTP APIs.
It reads each host's live OpenAPI contract and imports no maco server code, so
this directory builds on its own (Go version in `go.mod`; with the default
`GOTOOLCHAIN=auto`, `go` downloads it):

```sh
go build -o maco-mcp ./cmd/maco-mcp
go test ./...
```

Create an API key on each maco server, then write the client config. The
default path is `$XDG_CONFIG_HOME/maco/client.yml` (usually
`~/.config/maco/client.yml`); override it with `$MACO_CLIENT_CONFIG` or
`--config`.

```yaml
default: lab
instances:
  lab:
    url: https://maco.example.com:8080
    token_env: MACO_LAB_API_TOKEN
```

Set `MACO_LAB_API_TOKEN` in the MCP client's environment. For a trusted host
with a self-signed certificate, add `tls: {skip_verify: true}` to its entry.
Then register `/absolute/path/maco-mcp` as a command in your MCP client. MCP
messages go to stdout, diagnostics to stderr.

Ask the agent to call `instances_list`, then `instance_describe` with
`instance: lab` for the host's guide and tool list. Hosts must publish the
[version 1 contract](CONTRACT.md); a host that is down or too old reports an
error while the others keep working.

Full guide: [docs/mcp.md](../docs/mcp.md). Tests use a frozen API fixture;
the binary itself has no bundled contract.
