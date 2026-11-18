# Agent contract, version 1

`maco-mcp` fetches `/api/docs/openapi.json` from each configured host with its
bearer token. The document must be OpenAPI 3.1 and have a root `x-maco`:

```json
{"x-maco":{"version":1,"instructions":"Explain onboarding and resource workflows here."}}
```

## Operations

Each exposed operation needs an `operationId`, a useful `description`, and an
`x-maco` object:

```json
{"x-maco":{"expose":true,"readOnly":false,"transport":"json",
  "wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}}
```

- Paths are local `/api/` routes. External references, server URLs and header
  or credential overrides are not supported.
- Parameters are path or query parameters. A JSON request body becomes the MCP
  `body` argument. Schemas use local `#/components/schemas/` references.
- Schemas should document units, limits, examples, permissions and where IDs
  come from.
- Set `expose: false` for interactive sessions, login and unsupported
  transports.

## Transports

| `transport` | Use |
|-------------|-----|
| `json` | ordinary JSON request and response |
| `multipart` | file upload, with an `upload` object |
| `sse` | finite server-sent events, with a `stream` object |
| `image` | PNG response, returned as MCP image content |

Uploads declare the form field, the size limit and the accepted file suffixes.
The MCP tool takes an absolute `file_path` to a regular file on the MCP host:

```json
{"x-maco":{"expose":true,"readOnly":false,"transport":"multipart",
  "upload":{"field":"file","maxBytes":10737418240,"extensions":[".iso"]}}}
```

A stream declares its final event and how to read failure from it. Other events
are read until it arrives:

```json
{"x-maco":{"expose":true,"readOnly":true,"transport":"sse",
  "stream":{"event":"done","stateField":"state","failureValue":"failed"}}}
```

## Waiting

`wait` (optional) marks an action that returns a job. `idField` is the
top-level response field holding the job ID, which fills the target
operation's only path parameter. The target must be an exposed `GET` with the
`sse` transport. With MCP `wait: true` the tool submits the action and follows
the target; otherwise it returns the original response. A failed wait keeps the
job ID in its error. Cancelling ends the HTTP wait, not the server job.

Ordinary requests time out after 30 seconds; uploads and waits after two hours.

## Discovery

`instance_describe` returns the instructions, schemas, metadata, contract hash
and tool mapping. Hosts with identical contracts share tool names; differing
contracts get stable hash suffixes and separate instance enums. A host whose
discovery fails loses its action tools but keeps its status.
`instances_refresh` reloads contracts; clients should list tools again
afterwards.
