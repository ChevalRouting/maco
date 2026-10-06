# Maco MCP

An independent Go module providing a stdio MCP client for one or more Maco
servers. It uses their live OpenAPI contracts and imports no Maco server code.
You can copy this directory alone to build it (Go 1.25 or newer):

```sh
go build -o maco-mcp ./cmd/maco-mcp
go test ./...
```

Create an API key on your Maco server and write the shared client config.
By default `maco-mcp` reads `$XDG_CONFIG_HOME/maco/client.yml` (usually
`~/.config/maco/client.yml`), overridable with `$MACO_CLIENT_CONFIG` or
`--config`. This is the same file the `infra/` Ansible tooling consumes:

```yaml
default: lab
instances:
  lab:
    url: https://maco.example.com:8080
    token_env: MACO_LAB_API_TOKEN
```

Set that environment variable in your MCP client's process environment. For
trusted self-signed hosts, add `tls: {skip_verify: true}` to that instance.
Launch the binary (reads the default path):

```sh
/absolute/path/maco-mcp
```

Register this command and its arguments in your MCP client. Protocol output
uses stdout and diagnostics use stderr. The executable needs no source files,
Go installation, Task, or local VM service at runtime.

Ask the agent to call `instances_list`, then `instance_describe` for `lab`.
The response explains workflows, permissions, resource discovery, and available
operations. Pass `operation: createVM` to inspect its complete request and
response contract before calling its advertised tool. Async actions accept
`wait: true`; without it, retain the returned job ID and call the advertised
wait tool before depending on completion. Failed jobs are MCP errors.

Maco servers must publish the [version 1 agent contract](CONTRACT.md). Upgrade
older servers first. Offline or incompatible hosts report discovery errors;
other configured hosts remain usable. Call `instances_refresh` after upgrades
or recovery and re-list tools. When host contracts differ, use the per-instance
operation-to-tool mapping instead of guessing suffixed names.

Uploads read absolute file paths on the machine running this executable.
Interactive WebSocket sessions and continuous subscription streams are excluded.
Tests use a frozen API fixture only; production discovery has no bundled fallback.
