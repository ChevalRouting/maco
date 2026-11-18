# Security

Read this before exposing maco beyond a trusted network. Accounts and roles are
in [authentication.md](authentication.md).

## An admin account is root on the host

- The daemon runs as root, because it manages host network interfaces.
- The web UI has a **Console** that opens a login shell on the host
  (`/api/host/console`). It runs with the daemon's privileges, so on an
  installed maco it is a root shell. Only `admin` accounts can open it.
- Admins can also change host networking, attach USB devices and power off the
  host.

Treat admin credentials (and admin API keys) as root passwords, and keep the
API on a trusted network.

## QEMU runs as root

QEMU is started by the root daemon in its own session, with no separate user
and no sandbox profile, so a guest that escapes QEMU gets root on the host. Do
not run untrusted guests, and prefer a dedicated host. Running QEMU as an
unprivileged user or under `sandbox-exec` is planned.

## Secrets in the database

`web.db` holds job payloads and history, API keys and password hashes. Guest
setup supplied when a VM is created (which may include passwords) passes
through a job payload. maco creates the data directory readable only by its
owner (root for the daemon). Finished queue messages are deleted, but treat
`web.db` and its backups as secrets. Prefer SSH keys to guest passwords.

## Transport

- The UI and API are HTTPS by default, with a self-signed certificate unless
  you configure one (see
  [configuration.md](configuration.md#tls-certificates)).
- The browser stores its JWT in `localStorage` and sends it as a bearer header.
  WebSockets receive it in their first message and check the origin. Tokens
  never go in URLs.
- Login tokens expire after 24 hours. API keys do not expire: revoke the ones
  you no longer use.
- Responses carry `Content-Security-Policy`, `X-Frame-Options`,
  `X-Content-Type-Options` and `Referrer-Policy`.

## Image integrity

A downloaded cloud image is checked only when its catalog entry has a `SHA256`.
The built-in entries point at each distribution's moving "latest" image, so they
have no hash and are not verified. To verify, use a versioned image URL with
its published checksum, or check uploaded images yourself.
