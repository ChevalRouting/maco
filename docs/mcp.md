# MCP server

`maco-mcp` is a standalone stdio MCP server for agents managing one or more
Maco hosts. It follows Routier's named-instance registry and bearer-token
model. It runs on the agent's machine and connects to existing Maco HTTP APIs;
it does not start a Maco daemon or require local QEMU, Redis, or root access.

## Build and configure

```sh
task build-mcp
cp maco-mcp.example.yml maco-mcp.yml
chmod 600 maco-mcp.yml
```

Create an API key in Maco's API page, then configure the registry:

```yaml
instances:
  lab:
    url: https://maco.example.com:8080
    token_env: MACO_LAB_API_TOKEN
    tls:
      skip_verify: false
  desktop:
    url: https://localhost:8080
    token_env: MACO_DESKTOP_API_TOKEN
    tls:
      skip_verify: true
```

`token` may be used instead of `token_env`, but not together. Environment
variables must be present in the MCP process environment. API keys inherit the
account's permissions, including the server's admin/viewer restrictions. A
login JWT also works until it expires. Use `skip_verify: true` only for a trusted
host with a self-signed certificate, such as Maco's default local certificate.
TLS verification is enabled by default. HTTP redirects are rejected.

The local `maco-mcp.yml` registry is ignored by Git. The checked-in example
contains no credentials. Instance discovery returns names and contract status, and tool
arguments never override registry URLs or authentication headers. Account
operations such as `createAPIKey` can return newly created secrets as their
normal API response.

Configure your MCP client to launch:

```sh
/path/to/maco-mcp --config /absolute/path/to/maco-mcp.yml
```

For clients using a JSON MCP registry:

```json
{
  "mcpServers": {
    "maco": {
      "command": "/absolute/path/to/maco-mcp",
      "args": ["--config", "/absolute/path/to/maco-mcp.yml"]
    }
  }
}
```

Supply the token environment variables through your client's environment or
secret configuration. Protocol messages use stdout; diagnostics use stderr.

## Tools and arguments

Start with `instances_list`, then `instance_describe` with an `instance` name.
The latter returns the server's usage guide and operation-to-tool mapping.
Add `operation` to inspect an operation's complete request and response schemas.
`instances_refresh` reloads one or all hosts after upgrades or connection failures;
re-list tools afterward.

Action tools normally use the API's OpenAPI `operationId` as their name and take
an `instance` argument. When hosts publish conflicting contracts, tools receive
stable hash suffixes; use the mapping from `instance_describe`. Path and
query parameters are top-level arguments, JSON request payloads go in `body`,
and multipart uploads take `file_path`.

Actions returning an asynchronous job also accept `wait` (default `false`).
With `wait: true`, the tool submits the action, extracts the returned job ID,
calls `/api/jobs/{id}/wait`, and returns the final job with its result or error.
With `wait: false`, it returns the queued job immediately. If waiting is
interrupted, the error retains the submitted job ID so the agent can resume
with `waitJob` or inspect it with `getJob` without repeating the action.

The client fetches `/api/docs/openapi.json` from each host and builds validated
tool schemas from that live contract, including nested request types. Maco must
publish the version 1 `x-maco` agent metadata introduced with this implementation;
upgrade the Maco server first. Older or unreachable hosts show discovery errors
while healthy hosts remain usable. There is no embedded contract fallback.

The independent Go module lives in `mcp/`; it imports no Maco server packages.
You can copy that directory alone and build with
`go build -o maco-mcp ./cmd/maco-mcp`. The resulting executable runs from any
working directory with just its registry and credentials. Go, the checkout,
Task, and QEMU are not runtime requirements. See [the module README](../mcp/README.md).

| Area | Representative tools |
|------|----------------------|
| Host and status | `hostInfo`, `listVMs`, `getVM`, `vmGuestAgent`, `previewVM` |
| VM lifecycle | `createVM`, `startVM`, `shutdownVM`, `stopVM`, `deleteVM` |
| VM configuration | `updateHardware`, `getGuestSetup`, `updateGuestSetup`, `previewGuestSetup`, `updateMedia` |
| Backups | `createBackup`, `listBackups`, `restoreBackup`, `deleteBackup`, `getBackupSchedule`, `setBackupSchedule` |
| Snapshots | `createSnapshot`, `listSnapshots`, `restoreSnapshot`, `deleteSnapshot` |
| Networking | `listNetworks`, `createNetwork`, `updateNetwork`, `applyNetwork`, `destroyNetwork`, `listInterfaces`, `suggestMAC`, `addVMInterface`, `updateVMInterface`, `removeVMInterface` |
| Storage and media | `listDisks`, `diskStorage`, `addDisk`, `growDisk`, `removeDisk`, `listMedia`, `createMedia`, `uploadISO`, `uploadImage`, `deleteMedia` |
| Images | `listImages`, `listCatalog`, `downloadCatalogImage`, `deleteCatalogImage` |
| USB | `listUSBDevices`, `listVMUSB`, `attachUSB`, `detachUSB`, `assignUSB`, `unassignUSB` |
| Jobs | `listJobs`, `getJob`, `waitJob` |
| Accounts | `currentUser`, `changePassword`, `listAPIKeys`, `createAPIKey`, `revokeAPIKey` |
| Host power | `powerOffHost`, `rebootHost` |

`previewVM` returns MCP image content. Uploads stream a regular file from the
MCP server's filesystem, using an absolute path; file bytes are not embedded in
tool arguments. ISO and disk-image uploads follow the API's 10 GiB and 64 GiB
limits. Ordinary requests time out after 30 seconds; uploads and `waitJob` allow
up to two hours. Client cancellation cancels the HTTP request, but an already
submitted server job can continue running.

Interactive WebSocket host shells, guest serial consoles, graphical displays,
and continuous event/job subscription streams are not exposed. Use the Maco UI
for these. Login is handled by the preconfigured registry credential instead
of a tool. `waitJob` consumes the API's finite SSE stream until completion.

## Example workflow

List VM status with `listVMs`:

```json
{"instance":"lab"}
```

Discover available images and networks, then call `createVM` with appropriate
values from those results:

```json
{
  "instance":"lab",
  "body":{
    "name":"agent-vm",
    "cpus":2,
    "memory_mib":2048,
    "disk_size_gib":20,
    "username":"admin",
    "image":"ubuntu-24.04-arm64"
  }
}
```

Add `"wait": true` alongside `instance` and `body` to wait within the creation
tool. Otherwise, the response is a queued job, not proof that creation succeeded. Pass its ID to
`waitJob`, then use `listVMs` or `getVM` to inspect the resulting VM:

```json
{"instance":"lab","id":"<job-id>"}
```

Create a backup with `createBackup`:

```json
{"instance":"lab","id":"<vm-id>","wait":true}
```

This waits for the backup job and returns its final state. Then call `listBackups`. To restore a backup as a new VM,
call `restoreBackup` with:

```json
{
  "instance":"lab",
  "id":"<vm-id>",
  "timestamp":"<timestamp-from-listBackups>",
  "body":{"as_new":true}
}
```

HTTP failures and failed jobs returned by `waitJob` become MCP tool errors.
Read-only operations carry the MCP read-only annotation. Mutating operations
retain conservative destructive defaults; the API remains authoritative for
validation and permissions. Deletion, in-place restore, forced stop, and host
power operations act on the selected host immediately or enqueue real jobs.
