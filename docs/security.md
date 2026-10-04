# Security posture

This page summarizes maco's trust model and the known limitations to weigh
before exposing it beyond a trusted operator. Accounts and roles are covered in
[authentication.md](authentication.md).

## Runs as root

maco manages host networking (native bridges, feth pairs, VLAN interfaces) and
privileged VM networking directly, so the daemon runs as root. Any authenticated
`admin` account therefore has root-equivalent reach over the host. Treat admin
accounts accordingly and keep the API on a trusted network.

## QEMU runs as root (isolation limitation)

QEMU processes are started by the root daemon in a new session but without a
dedicated unprivileged uid or a sandbox profile, so guest device emulation runs
as root. A QEMU escape would therefore run as root. This is a known first
release limitation. Do not run untrusted guests, and prefer a dedicated host.
Running QEMU under a lower-privilege uid or a `sandbox-exec` profile is a planned
hardening.

## Host console

The web UI exposes a **Console** that opens a login shell on the maco host over
an authenticated WebSocket (`/api/host/console`). The shell runs with the
daemon's own privileges, so on a production install it is a root shell on the
host. It is restricted to `admin` accounts and enforced on the server, the same
way the interactive VM consoles are. This makes the host-root reach of an admin
account explicit: treat admin credentials as root on the host, and keep the API
on a trusted network.

## Redis

Job payloads and history live in Redis. When maco manages the bundled Redis
daemon it binds loopback only and sets a generated `requirepass` password,
stored at `<data-dir>/redis.secret` (0600) and threaded into the daemon's Redis
URL, so other local accounts cannot read task payloads or forge queued
operations. When you point maco at an external Redis with `MACO_REDIS_URL`,
supply credentials in that URL and restrict access yourself. The development
`compose.yml` binds loopback without a password; add `--requirepass` for
any shared host.

VM passwords supplied at creation transit the job payload in Redis. The
`requirepass` above protects them at rest from other local accounts; prefer SSH
keys over passwords where possible.

## Transport

The API and UI are served over HTTPS by default (`maco serve --tls`, self-signed
in the data directory unless you pass `--tls-cert`/`--tls-key`). The browser
holds a JWT in `sessionStorage` and sends it as a bearer header; the console and
event WebSockets authenticate with the token in their first message and enforce a
same-origin check. Responses carry `Content-Security-Policy`, `X-Frame-Options`,
`X-Content-Type-Options`, and `Referrer-Policy`.

## Cloud-image integrity

Downloaded cloud images are checksum-verified when a catalog entry carries a
`SHA256`. The built-in catalog entries track each distribution's moving "latest"
image, whose contents change over time, so they intentionally leave `SHA256`
empty rather than pin a hash that upstream will rotate. Pin a versioned image URL
with its published `SHA256` to enable verification, or verify uploaded images out
of band.
