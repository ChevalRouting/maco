# Agent contract, version 1

The MCP client fetches `/api/docs/openapi.json` from each configured registry URL
using its bearer credential. The response must be OpenAPI 3.1 and include:

```json
{"x-maco":{"version":1,"instructions":"Explain onboarding and resource workflows here."}}
```

Each exposed operation requires an `operationId`, a useful `description`, and
explicit transport metadata. Paths are local `/api/` routes; external references,
remote server URLs, and credential/header overrides are unsupported. Request
schemas use local `#/components/schemas/` references. Parameters may be path or
query parameters. JSON request bodies become the MCP `body` argument. Schemas
should document units, constraints, examples, permissions and ID discovery.

```json
{"x-maco":{"expose":true,"readOnly":false,"transport":"json",
  "wait":{"operationId":"waitJob","idField":"id","parameter":"id"}}}
```

`wait` is optional and declares an asynchronous action. Its top-level response
`idField` supplies the job identifier for the target operation's sole path
parameter. The target must be an exposed GET with the finite SSE transport.
Setting MCP `wait: true` submits the action and follows that operation; otherwise
the original response is returned. A wait error retains the submitted job ID.

```json
{"x-maco":{"expose":true,"readOnly":true,"transport":"sse",
  "stream":{"event":"done","stateField":"state","failureValue":"failed"}}}
```

The terminal SSE event contains a JSON object. Its state identifies failure;
other events are consumed until termination. Cancellation ends the HTTP wait,
not the server job. Ordinary requests have a 30-second timeout and uploads and
waits have a two-hour timeout.

Multipart operations declare the form field, byte limit, and accepted suffixes:

```json
{"x-maco":{"expose":true,"readOnly":false,"transport":"multipart",
  "upload":{"field":"file","maxBytes":10737418240,"extensions":[".iso"]}}}
```

They accept an absolute `file_path` referring to a regular file on the MCP host.
`transport: image` supports PNG responses as MCP image content. Other supported
transport values are `json`, `multipart`, and finite `sse`. Set `expose: false`
for interactive sessions, authentication, or unsupported transports.

`instance_describe` publishes instructions, request/response schemas, metadata,
contract hash and tool mapping. Identical contracts share operation names;
conflicting contracts use stable hash suffixes and separate instance enums.
Discovery failure removes that instance's action tools without hiding its status.
`instances_refresh` reloads contracts; clients should re-list tools afterward.
