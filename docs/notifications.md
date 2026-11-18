# Events and notifications

The web UI keeps one WebSocket open to `/api/events` and uses it to learn when
jobs change and when to reload data. It is plain WebSocket with a JSON
protocol (not Socket.IO), served by `github.com/coder/websocket`.

## Connecting

1. Open `wss://<host>/api/events` from the same origin.
2. Send `{"token": "<JWT>"}` as the first message within five seconds. Use a
   login token: API keys are not accepted here.
3. A bad token closes the socket with code `1008`, and so does the token
   expiring or being revoked (checked every five seconds) while connected. If
   the server cannot check the token, it closes with `1013`.

Any role can connect. The server sends WebSocket pings to keep the connection
alive.

`/api/jobs/stream` works the same way but only sends job messages
(`snapshot` and `update`).

## Messages

| Type | Content |
|------|---------|
| `snapshot` | `jobs`: the current public jobs (may be omitted when empty). Sent on every connect, so a reconnect recovers the full state. |
| `update` | `job`: one public job, with its state and logs. |
| `invalidate` | `resources`: changed areas (`vms`, `networks`, `media`, `disks`, `storage`, `interfaces`, `usb`), `preview:<vm-uuid>` for a new screenshot, or `all` to reload everything. |

The message type is `JobsEvent` in `pkg/api/job_stream.go`. Queue payloads and
guest credentials are never sent, and private jobs (preview captures) are left
out.

## Where events come from

- **Jobs**: every job change writes a row to `job_events` in the same
  transaction. Each connection polls that table every 250 ms.
- **Finished mutations**: when a job or a media change finishes, the server
  invalidates the areas it touched.
- **Outside changes** (CLI edits, VM exits, USB hotplug, disk usage): while at
  least one client is connected, the server checks state every two seconds and
  invalidates what changed.
- A client too slow to keep up gets `all` instead of missing changes.

## In the browser

- Each page loads its data over HTTP once, then reloads only when a matching
  `invalidate` arrives. A change that arrives during a load triggers another
  load, so a late response never hides a newer change.
- Preview images are fetched over HTTP when a `preview:` event arrives.
- The jobs panel and jobs page share the same connection. Job logs are loaded
  from `GET /api/jobs/{id}`.
- On disconnect the browser reconnects with a delay between one and fifteen
  seconds, then reloads what is on screen after the new snapshot.
