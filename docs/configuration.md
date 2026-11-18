# Configuration

Each VM and each network is one YAML manifest in the data directory. The
manifests are the source of truth; the SQLite database never overrides them.
Manifests are written with mode `0600`. You can edit them by hand (or with
`maco vm edit` / `maco networks edit`); changes apply the next time maco acts
on that VM or network.

## VM manifest

`<data-dir>/vms/<uuid>.yml`:

```yaml
id: 20d20d55-0370-4beb-a89c-8cdc03b607d0
name: web1
image: ubuntu-24.04-arm64
cpus: 2
memory_mib: 2048
disk_size_gib: 20
network: user
ssh_keys:
  - ssh-ed25519 AAAA... admin@example.com
tags: [web, prod]
autostart: true
```

| Field | Notes |
|-------|-------|
| `id` | UUID set by maco; names the manifest and the disk directory |
| `name` | RFC 1123 hostname, also the default guest hostname |
| `image` | catalog image key (`maco image list`) or uploaded disk image |
| `isos` | media IDs of attached ISOs, used instead of `image` for an ISO install |
| `boot_order` | entries like `disk`, `disk:<disk-id>`, `iso:<media-id>` |
| `cpus` | vCPUs, at least 1 |
| `memory_mib` | memory in MiB, at least 64 |
| `disk_size_gib` | boot disk size in GiB |
| `network` | network name or UUID; empty or `user` means user-mode NAT |
| `addresses` | static IPv4/IPv6 CIDRs for the primary interface; DHCP otherwise |
| `gateway`, `nameservers` | optional, for the primary interface |
| `interfaces` | explicit interface list (`id`, `network`, `mac`, `addresses`, `gateway`, `nameservers`); replaces `network` |
| `disks` | data disks (`id`, `name`, `size_gib`) |
| `usb` | saved USB assignments, see [usb.md](usb.md) |
| `ssh_keys` | public keys available to guest setup templates |
| `tags` | free-form labels for grouping and filtering |
| `guest_setup` | first-boot configuration, see below |
| `autostart` | start on `maco reconcile` and when the daemon starts |

The manifest has no desired power state: `maco vm start` and `stop` are the
controls, and `autostart` only matters at reconcile. The runtime state (phase,
pid, boot time) lives in `web.db` and is rebuilt from the running QEMU
processes by `maco vm list` and `maco reconcile`.

## Guest setup

`guest_setup` configures the guest on first boot. maco writes it to a
cloud-init NoCloud seed ISO or an Ignition config attached to the VM.

```yaml
guest_setup:
  provisioner: cloud-init
  mode: raw
  raw: |
    #cloud-config
    hostname: [[ .Name ]]
    users:
      - name: maco
        sudo: "ALL=(ALL) NOPASSWD:ALL"
        ssh_authorized_keys:
    [[- range .SSHKeys ]]
          - [[ . ]]
    [[- end ]]
```

| Field | Notes |
|-------|-------|
| `provisioner` | `cloud-init`, `ignition` or `none` (no seed at all) |
| `mode` | `raw` |
| `raw` | the cloud-init `user-data`, or the Ignition Butane config (`config.bu`) |
| `files` | instead of `raw`: a list of `name` and `content` files to put in the seed |
| `hostname` | guest hostname; defaults to `name` |

`raw` and file contents are Go templates with `[[ ]]` delimiters. Available
values: `.Name`, `.Hostname`, `.CPUs`, `.MemoryMiB`, `.DiskSizeGiB`, `.SSHKey`
(the first key), `.SSHKeys`, `.Network`, `.Addresses`, `.Gateway`,
`.Nameservers`, `.Disks` and `.Interfaces`. maco also writes a `network-config`
for cloud-init from the VM's interfaces.

The web UI's default templates (`ubuntu-cloud-init`, `flatcar-podman`) are
complete examples. `maco vm new --guest-config FILE` stores FILE as raw guest setup; without it the VM gets `provisioner: none`.

## Network manifest

`<data-dir>/networks/<uuid>.yml`. `maco networks apply` makes the host match it.

```yaml
id: 214125b7-1ba8-47db-9796-708952310ed7
name: lab
mode: bridge
address: 192.0.2.1/24
members: [en1]
vlans:
  - parent: en0
    tag: 123
```

| Field | Notes |
|-------|-------|
| `mode` | `bridge`, `vmnet-bridged`, `user`, `switch` or `vlan` (`bridged` is an old name for `vmnet-bridged`) |
| `members` | existing host interfaces to add to the bridge |
| `vlans` | VLAN interfaces maco creates on a parent and adds to the bridge |
| `address` | host IPv4 CIDR on the bridge; no DHCP, NAT, DNS or routing comes with it |
| `device` | an existing bridge to use; maco leaves it in place on destroy |
| `uplink` | physical interface for `vmnet-bridged` |
| `parent`, `tag` | for `vlan` mode |
| `group` | internal multicast group for `switch` mode |

- A bridge does no VLAN filtering between its ports, so use one bridge per
  access VLAN.
- Members, VLANs and the address change live, even while VMs use the bridge.
- The uplink of a `vmnet-bridged` network is bound when QEMU starts, so stop
  every VM on it before changing it.
- maco records what it created in `owned`, `applied_members`, `applied_vlans`,
  `applied_address` and the VLAN `device` values. Do not edit these. maco never
  destroys interfaces it did not create.
- The CLI stores a VM's network by UUID, so renaming a network does not detach
  its VMs.

See [network-modes.md](network-modes.md) for what each mode does.

## TLS certificates

The web UI and API always serve HTTPS. Without a `tls` section in
`<data-dir>/config.yaml`, maco serves the self-signed `tls.crt` and `tls.key`
it generates in the data directory.

Admins can change the certificate in the web UI (**Certificates**) or with
`PUT /api/tls`. The change is validated, applied without a restart and saved to
`config.yaml`; uploaded files go to `<data-dir>/tls/`. Hand edits to
`config.yaml` apply at the next `maco serve` start (for the daemon,
`sudo maco service install`).

The `--tls-cert` and `--tls-key` flags of `maco serve`, `maco install` and
`maco service install` override the whole `tls` section, and the web UI then
shows the certificate as read-only. `cert`/`key` and `acme` cannot be combined.

### Your own certificate

```yaml
tls:
  cert: /etc/ssl/maco/fullchain.pem
  key: /etc/ssl/maco/privkey.pem
```

maco reloads the files when they change, so an externally renewed certificate
is picked up without a restart. If the new files fail to load, it keeps the old
certificate and logs a warning.

### ACME

```yaml
tls:
  acme:
    directory: https://ca.example.internal/acme/acme/directory
    email: ops@example.com
    domains: [maco-1.example.com]
    http_addr: ":80"
    ca: /etc/ssl/internal-root.pem
    eab_kid: kid-from-the-ca
    eab_key: base64url-mac-key
```

| Field | Notes |
|-------|-------|
| `directory` | ACME directory URL; default Let's Encrypt production |
| `email` | account contact, optional |
| `domains` | names to request, required |
| `http_addr` | listener for http-01 challenges, default `:80`; other requests redirect to HTTPS |
| `ca` | extra PEM roots for talking to the ACME directory (a private CA such as step-ca) |
| `eab_kid`, `eab_key` | external account binding, if the CA requires it |

- Only HTTP-01 is supported: the CA must reach each domain on port 80 at
  `http_addr`. The HTTPS listener can use any port.
- maco requests certificates at startup and when the settings change, renews
  them 30 days before expiry, and retries failures every 15 minutes.
- Account keys and certificates are cached in `<data-dir>/acme/`, one
  directory per ACME directory URL.
- Connections whose SNI matches an issued domain get the ACME certificate.
  Everything else (pending or failed domains, IP addresses, `localhost`) gets
  the self-signed one, so the UI stays reachable.
- The UI shows each domain's last error and a banner if the challenge listener
  cannot bind. The service log records every attempt.

`maco tls status` lists the certificates, their issuer and expiry, and
`pending` for ACME domains without a certificate yet.
