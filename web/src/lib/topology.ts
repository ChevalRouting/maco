import { type Network, type NetworkInterface, type VMView } from '../api'
import { networkModeName } from '../ux'

export interface TopologyPort {
  device: string
  detail?: string
  up?: boolean
  hardware?: string
}

export interface TopologyVM {
  id: string
  name: string
  running: boolean
}

export interface TopologyRow {
  key: string
  name: string
  mode: string
  device?: string
  network?: Network
  ports: TopologyPort[]
  vms: TopologyVM[]
}

export interface Topology {
  rows: TopologyRow[]
  unused: NetworkInterface[]
}

function vmRefs(vm: VMView): string[] {
  const interfaces = vm.manifest.interfaces
  if (interfaces?.length) return interfaces.map((nic) => nic.network || 'user')
  return [vm.manifest.network || 'user']
}

function attached(vms: VMView[], matches: (ref: string) => boolean): TopologyVM[] {
  return vms
    .filter((vm) => vmRefs(vm).some(matches))
    .map((vm) => ({ id: vm.manifest.id, name: vm.manifest.name, running: vm.phase === 'running' }))
    .sort((a, b) => a.name.localeCompare(b.name))
}

function networkPorts(network: Network, networks: Network[], hostPorts: NetworkInterface[]): TopologyPort[] {
  const up = (device: string) => hostPorts.find((port) => port.device === device)?.up
  const hardware = (device: string) => hostPorts.find((port) => port.device === device)?.hardware_port || undefined
  const vlanDevice = (device: string) => networks.find((item) => item.mode === 'vlan' && item.device === device)
  if (network.mode === 'bridge') {
    const members = (network.members ?? []).map((device) => {
      const vlan = vlanDevice(device)
      return vlan
        ? { device: vlan.parent || device, detail: `VLAN ${vlan.tag} · ${device}`, up: up(vlan.parent || ''), hardware: hardware(vlan.parent || '') }
        : { device, up: up(device), hardware: hardware(device) }
    })
    const vlans = (network.vlans ?? []).map((vlan) => ({ device: vlan.parent, detail: `VLAN ${vlan.tag}`, up: up(vlan.parent), hardware: hardware(vlan.parent) }))
    return [...members, ...vlans]
  }

  if (network.mode === 'vmnet-bridged' || network.mode === 'bridged') {
    return network.uplink ? [{ device: network.uplink, up: up(network.uplink), hardware: hardware(network.uplink) }] : []
  }

  if (network.mode === 'vlan' && network.parent) {
    return [{ device: network.parent, detail: `VLAN ${network.tag}`, up: up(network.parent), hardware: hardware(network.parent) }]
  }

  return []
}

export function topologyRow(network: Network, networks: Network[], vms: VMView[], hostPorts: NetworkInterface[]): TopologyRow {
  return {
    key: network.id,
    name: network.name,
    mode: networkModeName(network.mode),
    device: network.device || undefined,
    network,
    ports: networkPorts(network, networks, hostPorts),
    vms: attached(vms, (ref) => ref === network.id || ref === network.name),
  }
}

export function buildTopology(networks: Network[], vms: VMView[], hostPorts: NetworkInterface[]): Topology {
  const rows: TopologyRow[] = []
  const userVMs = attached(vms, (ref) => ref === 'user')
  if (userVMs.length) rows.push({ key: 'user', name: 'User NAT', mode: networkModeName('user'), ports: [], vms: userVMs })

  for (const network of [...networks].sort((a, b) => a.name.localeCompare(b.name))) rows.push(topologyRow(network, networks, vms, hostPorts))

  const used = new Set(rows.flatMap((row) => row.ports.map((port) => port.device)))
  return { rows, unused: hostPorts.filter((port) => port.hardware_port && !used.has(port.device)) }
}
