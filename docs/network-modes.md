# Network modes

A VM created without a network gets outbound internet through user-mode NAT.
For VMs that talk to each other or sit on your LAN, create a named network and
attach VMs to it:

```sh
maco networks new lab --mode bridge --member en0
maco vm new guest --network lab
```

| Mode | The VM reaches | Changes host interfaces | Use it when |
|------|----------------|-------------------------|-------------|
| `user` (default) | the internet, outbound only | no | the VM just needs to be online |
| `switch` | other VMs on the same switch | no | you want a private network between VMs |
| `bridge` | your LAN, or one VLAN on it | yes | the VM should look like a real machine on the network |
| `vmnet-bridged` | your LAN, through Apple's vmnet | yes | you prefer Apple's bridging on a Wi-Fi or Ethernet port |
| `vlan` | nothing by itself | yes | you want to manage a VLAN interface and add it to a bridge yourself |

The Networks page of the web UI draws each network as a switch, with its VMs on
one side and its host interfaces or VLANs on the other, plus run and link
state. Interfaces no network uses are listed at the end.

## User NAT

```sh
maco vm new guest --network user
```

The guest gets an address and DNS by DHCP and reaches the internet through the
Mac. Nothing on your LAN can connect to it. Each interface has its own private
NAT, so two VMs on `user` cannot see each other; use a `switch` for that. You
can also create a named network with `--mode user`.

## Switch

```sh
maco networks new inside --mode switch
maco vm new a --network inside
maco vm new b --network inside
```

VMs on the same switch share a private Ethernet segment. There is no link to
the LAN or the internet, and no DHCP or router: give guests static addresses or
run those services in one of the VMs.

## Bridge

```sh
# untagged, on a physical port
maco networks new office --mode bridge --member en0

# on VLAN 123, carried by en0
maco networks new vlan123 --mode bridge --vlan en0:123
```

The guest joins a real Ethernet segment and gets its address from the DHCP
server already there. With `--vlan en0:123` the guest sends untagged frames and
maco tags them on `en0`; the physical switch port must carry VLAN 123.

The bridge itself has no DHCP, DNS, routing or NAT. You can give the host an
address on it with `--address 192.0.2.1/24`.

To manage the VLAN interface yourself, create it with
`--mode vlan --parent en0 --tag 123`, then add the resulting `vlanN` device to a
bridge with `--member vlanN`. A `vlan` network cannot be attached to a VM
directly.

## vmnet bridged

```sh
maco networks new physical --mode vmnet-bridged --uplink en0
maco vm new guest --network physical
```

This uses Apple's vmnet bridged backend on the uplink, and the guest gets its
address from your LAN. vmnet decides which interfaces it accepts: maco only
checks that the uplink exists when you create the network, and vmnet attaches
when the VM starts. `bridged` is accepted as an old name for this mode.

## How bridges work

For `bridge` networks maco builds the plumbing on the host, so QEMU needs no
network privileges of its own:

1. a feth (virtual Ethernet) pair, with one end added to a macOS bridge next to
   your uplink or VLAN interface,
2. a small switch process, `maco-net-helper`, on the other end, which every VM
   on the network plugs into.

Traffic between VMs stays inside the switch; traffic to the host or the LAN
goes through the bridge. The switch opens its BPF descriptor as root, then
drops privileges. Destroying a network removes only what maco created.

`user` and `switch` run entirely inside QEMU and change nothing on the host.

## Changing interfaces on a running VM

You can add, change or remove a VM's interfaces without rebooting, from the web
UI or the interfaces API. maco hotplugs the device and saves the manifest only
after the change succeeds. If it cannot confirm the result, it reports that
recovery is needed instead of saving a manifest that does not match the VM.

- maco does not configure a hotplugged interface inside the guest. Set up DHCP
  or a static address there yourself.
- A VM created by an older maco needs one stop and start before hotplug works.
