# MCP server

`maco-mcp` is a stdio MCP server that lets an AI agent manage one or more maco
hosts through their HTTP APIs. It runs on the agent's machine and needs no
QEMU, root access or local maco daemon.

## Build

```sh
task build-mcp        # writes ./maco-mcp
```

The server is its own Go module in `mcp/` and imports no maco server code, so
you can also copy that directory alone and run
`go build -o maco-mcp ./cmd/maco-mcp`. The binary needs only its config file and
credentials at runtime. See [mcp/README.md](../mcp/README.md).

## Configure

`maco-mcp` reads the maco client config, shared with other maco client tools.
The default path is `$XDG_CONFIG_HOME/maco/client.yml` (usually
`~/.config/maco/client.yml`); override it with `$MACO_CLIENT_CONFIG` or
`--config`.

```sh
mkdir -p ~/.config/maco
cp maco-client.example.yml ~/.config/maco/client.yml
chmod 600 ~/.config/maco/client.yml
```

Create an API key on the API page of each maco web UI, then list the hosts:

```yaml
default: lab
instances:
  lab:
    url: https://maco.example.com:8080
    token_env: MACO_LAB_API_TOKEN
  desktop:
    url: https://localhost:8080
    token_env: MACO_DESKTOP_API_TOKEN
    tls:
      skip_verify: true
```

- `token_env` names an environment variable that must be set in the MCP
  process. `token` holds the key inline instead; use one or the other.
- A key has its account's role, so a `viewer` key gives read-only access.
- `skip_verify: true` turns off certificate checks. Use it only for a trusted
  host with maco's self-signed certificate. Verification is on by default.
- HTTP redirects are rejected.
- `default` is used by other client tools; `maco-mcp` ignores it and exposes
  every instance.

Register the binary in your MCP client:

```json
{
  "mcpServers": {
    "maco": {
      "command": "/absolute/path/to/maco-mcp",
      "args": ["--config", "/absolute/path/to/client.yml"]
    }
  }
}
```

Pass the token variables through the client's environment or secret settings.
MCP messages go to stdout, diagnostics to stderr.

## Using the tools

1. `instances_list` shows the configured hosts and whether their contract
   loaded.
2. `instance_describe` with an `instance` returns the host's usage guide and
   the operation-to-tool mapping. Add `operation` to get one operation's full
   request and response schema.
3. `instances_refresh` reloads one or all hosts after an upgrade or an outage.
   List the tools again afterwards.

Action tools are named after the API's `operationId` and take an `instance`
argument. Path and query parameters are top-level arguments, JSON bodies go in
`body`, and uploads take `file_path`. If hosts publish different contracts,
tool names get a hash suffix: use the mapping from `instance_describe`.

Tools are built from each host's live `/api/docs/openapi.json`, which must
carry the version 1 `x-maco` metadata ([contract](../mcp/CONTRACT.md)). There
is no built-in fallback: a host that is down or too old shows a discovery error
while the others keep working. Upgrade old servers first.

### Jobs and waiting

Actions that start a job accept `wait` (default `false`).

- `wait: false` returns the queued job right away. A queued job is not proof
  of success: pass its ID to `waitJob`, then check the resource.
- `wait: true` submits the action, waits on `/api/jobs/{id}/wait`, and returns
  the final job with its result or error.
- If waiting is interrupted, the error keeps the job ID. Resume with `waitJob`
  or `getJob` instead of repeating the action.

Failed HTTP calls and failed jobs become MCP tool errors.

### Limits

- Ordinary requests time out after 30 seconds; uploads and `waitJob` after two
  hours. Cancelling a tool call cancels the HTTP request, not a job already
  running on the server.
- Uploads read a regular file at an absolute path on the machine running
  `maco-mcp`. ISOs are limited to 10 GiB, disk images to 64 GiB.
- `previewVM` returns an MCP image.
- Not available: login, consoles, the host shell, and the WebSocket event
  streams. Use the web UI for those.
- Read-only operations carry the MCP read-only hint. Others are marked
  potentially destructive. Deleting, restoring in place, forced stop and host
  power act on the real host.

## Tools

| Area | Tools |
|------|-------|
| Host | `hostInfo`, `listInterfaces`, `powerOffHost`, `rebootHost`, `getTLS` |
| Metrics | `metricsHistory`, `metricInterfaces` |
| VMs | `listVMs`, `getVM`, `createVM`, `startVM`, `shutdownVM`, `stopVM`, `rebootVM`, `deleteVM`, `vmGuestAgent`, `previewVM` |
| VM settings | `updateHardware`, `getGuestSetup`, `updateGuestSetup`, `updateMedia` |
| Templates | `listTemplates`, `getTemplate`, `createTemplate`, `updateTemplate`, `deleteTemplate`, `instantiateTemplate` |
| Backups | `createBackup`, `listBackups`, `restoreBackup`, `deleteBackup`, `getBackupSchedule`, `setBackupSchedule`, `pruneBackups` |
| Snapshots | `createSnapshot`, `listSnapshots`, `restoreSnapshot`, `deleteSnapshot` |
| Networks | `listNetworks`, `createNetwork`, `updateNetwork`, `applyNetwork`, `destroyNetwork`, `suggestMAC`, `addVMInterface`, `updateVMInterface`, `removeVMInterface` |
| Disks and media | `listDisks`, `diskStorage`, `addDisk`, `growDisk`, `wipeDisk`, `replaceDisk`, `removeDisk`, `listMedia`, `createMedia`, `uploadISO`, `uploadImage`, `deleteMedia` |
| Images | `listImages`, `listCatalog`, `downloadCatalogImage`, `deleteCatalogImage` |
| USB | `listUSBDevices`, `listVMUSB`, `attachUSB`, `detachUSB`, `assignUSB`, `unassignUSB` |
| Jobs | `listJobs`, `getJob`, `waitJob` |
| Account | `currentUser`, `changePassword`, `listAPIKeys`, `createAPIKey`, `revokeAPIKey`, `listSSHKeys`, `addSSHKey`, `deleteSSHKey` |

`createAPIKey` returns the new secret in its response, like the API does.

## Example

List VMs with `listVMs`:

```json
{"instance": "lab"}
```

Check images with `listCatalog` and networks with `listNetworks`, then create a
VM with `createVM` and wait for it:

```json
{
  "instance": "lab",
  "wait": true,
  "body": {
    "name": "agent-vm",
    "image": "ubuntu-24.04-arm64",
    "cpus": 2,
    "memory_mib": 2048,
    "disk_size_gib": 20
  }
}
```

Back it up with `createBackup`, then confirm with `listBackups`:

```json
{"instance": "lab", "id": "<vm-id>", "wait": true}
```

Restore a backup as a new VM with `restoreBackup`:

```json
{
  "instance": "lab",
  "id": "<vm-id>",
  "timestamp": "<timestamp-from-listBackups>",
  "body": {"as_new": true}
}
```
