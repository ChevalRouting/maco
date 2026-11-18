# Authentication and roles

The web UI and the HTTP API use maco's own accounts, stored in the SQLite
database with bcrypt password hashes. They are not macOS user accounts.

## Accounts

```bash
maco user add alice --password secret --role admin
maco user add bob   --password secret --role viewer
maco user list
maco user passwd alice --password new
maco user rm bob
```

On first start, `maco serve` (and `maco install`) creates an `admin` account.
Its password comes from `MACO_ADMIN_PASSWORD`, or is generated and written to
the log once. A JWT signing key is created at `<data-dir>/jwt.secret` (mode
0600).

## Roles

`--role` defaults to `admin`.

| Role | Can do |
|------|--------|
| `admin` | Everything: create, change, start, stop and delete VMs, networks, disks, media and USB assignments; open the VM consoles and the host console. |
| `viewer` | Read only: list and inspect everything, watch job and event streams, see VM previews. No changes, no consoles. |

The server enforces roles on every request:

- The role is read from the database each time, not from the token, so a role
  change applies immediately.
- `GET` and `HEAD` are allowed for any account. `POST`, `PUT`, `PATCH` and
  `DELETE` need `admin`, otherwise `403`.
- The console, display and host console WebSockets need `admin`. The job and
  event streams are open to both roles.

`GET /api/me` returns `{"username": ..., "role": ...}`. The UI uses it to hide
actions a viewer cannot take, but the server still rejects them.

## Sessions

`POST /api/login` returns a JWT valid for 24 hours. Send it as
`Authorization: Bearer <token>`. WebSockets take it as their first message
(`{"token": "..."}`) within five seconds and check the origin. Tokens never go
in URLs.

The browser keeps the token in `localStorage` (or `sessionStorage` if
`localStorage` is unavailable), so it survives a reload or a server restart
until it expires. Because the signing key lives in the data directory, tokens
stay valid across restarts and upgrades.

- Changing a password invalidates that account's login tokens (not its API
  keys).
- A deleted or renamed account is rejected on its next request.
- An invalid, expired or revoked token gets `401` (WebSocket close code
  `1008`), and the browser signs out.
- If the user lookup fails for another reason (for example the server is
  shutting down), HTTP returns `503` and WebSockets close with `1013`, and the
  browser retries with the same token.

## API keys

For scripts and agents, create an API key on the API page of the web UI (or
`POST /api/me/api-keys`). The token starts with `maco_`, is shown once, and is
sent as a bearer token like a JWT. A key has the same role as its account and
works until you revoke it (`DELETE /api/me/api-keys/{id}`). WebSockets
(consoles and event streams) accept only login tokens, not API keys.

## Adding a role

Add a constant in `pkg/auth`, accept it in `auth.ValidRole`, and update the
permission check (`auth.CanMutate`). `/api/me` already exposes the role to the
UI.
