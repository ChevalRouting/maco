# API contract

The OpenAPI 3.1 spec is generated from the Go handlers: Swag reads the handler
annotations and request/response structs, and
[`@hey-api/openapi-ts`](https://heyapi.dev) generates the typed TypeScript
client the web UI uses.

## Using the API

- The spec is served at `/api/docs/openapi.json`, without login.
- Get a token with `POST /api/login` (or create an API key, see
  [authentication.md](authentication.md#api-keys)) and send
  `Authorization: Bearer <token>`.
- A `202` response holds a queued job, not the result. Follow
  `GET /api/jobs/{id}` or `GET /api/jobs/{id}/wait` for the outcome and logs.
- Console, display and event routes are WebSockets: send `{"token": "..."}` as
  the first message within five seconds. See
  [notifications.md](notifications.md).

## Regenerating

```sh
task api-tools          # install the pinned swag
npm ci --prefix web
task api-generate       # alias: task docs
task api-check
npm --prefix web run build
```

Commit `pkg/api/docs/swagger.json` and `web/src/api/generated/` with every
handler change. `task api-check` regenerates both into a temporary directory and
fails if they differ from the committed files, then checks that every mounted
route is in the spec and every spec operation is mounted. Run it in CI.

Writing handlers:

- Mark optional fields with `binding:"optional"`. Fields are required by
  default.
- `tools/openapi/normalize.py` fixes Swag's output after generation (multipart
  file fields, misplaced extensions, field descriptions). It holds no API
  paths or model names.
- `web/src/api.ts` re-exports generated types. Form types in the UI may make
  optional fields required locally.

## Agent metadata (`x-maco`)

The spec carries `x-maco` metadata that lets `maco-mcp` build its tools
without importing server code. The root has `version: 1` and onboarding
`instructions`; each operation declares whether it is exposed, read-only, its
transport, and optional upload or wait details. The format is specified in
[mcp/CONTRACT.md](../mcp/CONTRACT.md).

Where the text comes from:

- Operation guidance: the handler's `@Description` and `@x-maco` annotations.
- Field guidance: request struct tags (constraints, examples, and
  `extensions:"x-maco-description=..."`).
- Referenced types: their `@Description`.

Keep the root `@x-maco` annotation before the security annotations in
`pkg/api/doc.go`, or Swag parses it as a security extension.
