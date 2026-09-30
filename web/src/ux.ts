import type { Job, Network } from './api'

const actionNames: Record<string, string> = {
  'vm.create': 'Create Virtual Machine', 'vm.start': 'Start Virtual Machine',
  'vm.shutdown': 'Shut Down', 'vm.stop': 'Force Stop', 'vm.delete': 'Delete Virtual Machine',
  'vm.hardware': 'Save Hardware', 'vm.disk.add': 'Add Disk', 'vm.disk.grow': 'Increase Disk Capacity',
  'vm.disk.remove': 'Delete Disk', 'vm.interface.add': 'Add Network Adapter',
  'vm.interface.update': 'Save Network Adapter', 'vm.interface.remove': 'Remove Network Adapter',
  'vm.usb.attach': 'Connect USB Device', 'vm.usb.assign': 'Assign USB Device on Startup',
  'vm.usb.detach': 'Disconnect USB Device', 'vm.usb.unassign': 'Remove USB Assignment',
  'network.create': 'Create Network', 'network.apply': 'Apply Network',
  'network.update': 'Save Network', 'network.destroy': 'Delete Network',
  'vm.screenshot': 'Capture Screenshot', 'image.download': 'Download Image',
  'vm.autostart': 'Save Automatic Startup',
  'vm.backup': 'Back Up Virtual Machine', 'vm.backup.restore': 'Restore Backup',
  'vm.backup.delete': 'Delete Backup',
  'vm.snapshot.create': 'Create Snapshot', 'vm.snapshot.restore': 'Restore Snapshot',
  'vm.snapshot.delete': 'Delete Snapshot',
}

export function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / Math.pow(1024, exp)
  return `${Number(value.toFixed(exp === 0 ? 0 : 1))} ${units[exp]}`
}

export function activityName(job: Job): string {
  return actionNames[job.action] || job.label?.trim() || `Operation (${job.action || 'unknown'})`
}
export function activityState(job: Job): string {
  if (job.action === 'vm.shutdown' && job.state === 'succeeded') return 'Shutdown Requested'
  return { pending: 'Queued', running: 'In Progress', succeeded: 'Completed', failed: 'Failed' }[job.state] || 'Status Unavailable'
}
export function unfinishedTime(job: Job): string {
  return job.state === 'pending' ? 'Not Started' : job.state === 'running' ? 'In Progress' : 'Not Recorded'
}
export function memoryLabel(mib: number): string {
  return `${Number((mib / 1024).toFixed(10))} GiB`
}

export const automaticStartupHelp = 'Start stopped VMs when maco checks its configuration. A manually shut-down VM can start again at the next check.'
export const networkModes = [
  { value: 'user', label: 'Internet Access (NAT)', help: 'Outbound internet access with automatic addresses. Each adapter has a separate private network; other VMs and the local network cannot initiate connections to it.' },
  { value: 'switch', label: 'Isolated VM Network', help: 'VMs on this network can communicate with each other. No internet access or automatic addresses are supplied; configure guest addresses.' },
  { value: 'bridge', label: 'Host Bridge', help: 'Connect VMs to selected host interfaces, or to each other without an uplink. No automatic addresses or internet sharing are supplied. Connectivity and addressing depend on the attached network. Applies host configuration immediately.' },
  { value: 'vmnet-bridged', label: 'Local Network (Physical Adapter)', help: 'VMs join the local network through the selected adapter. Addresses and internet access come from that network. Requires an eligible adapter and administrator privileges on this Mac.' },
  { value: 'vlan', label: 'Tagged VLAN Interface', help: 'Create a tagged host interface for an existing VLAN. Add it to a host bridge to connect VMs. Addressing and internet access come from services on that VLAN; the physical network must carry the selected tag.' },
]
export const builtinNetworks = [
  networkModes[0],
]
export function networkModeName(mode: string): string {
  return networkModes.find(item => item.value === (mode === 'bridged' ? 'vmnet-bridged' : mode))?.label || mode
}
export function networkHelp(ref: string, networks: Network[]): string {
  const builtin = builtinNetworks.find(item => item.value === ref)
  const network = networks.find(item => item.id === ref || item.name === ref)
  return builtin?.help || networkModes.find(item => item.value === network?.mode)?.help || 'This network is unavailable. Choose another network before saving.'
}

export function cidrError(value: string, ipv4Only = false): string {
  if (!value.trim()) return ''
  const [address, prefix, ...extra] = value.trim().split('/')
  const v4 = address.split('.')
  const isV4 = v4.length === 4 && v4.every(part => /^\d{1,3}$/.test(part) && Number(part) <= 255)
  const halves = address.split('::')
  const groups = address.split(':').filter(Boolean)
  const isV6 = !ipv4Only && address.includes(':') && halves.length <= 2 && groups.every(part => /^[0-9a-f]{1,4}$/i.test(part)) && (halves.length === 2 ? groups.length < 8 : groups.length === 8)
  if (extra.length || (!isV4 && !isV6) || !/^\d{1,3}$/.test(prefix || '') || Number(prefix) > (isV4 ? 32 : 128)) {
    return ipv4Only ? 'Enter an IPv4 address and prefix, for example 192.168.1.20/24.' : 'Enter an IP address and prefix, for example 192.168.1.20/24 or 2001:db8::20/64.'
  }
  return ''
}
export function macError(value: string): string {
  return !value.trim() || /^([0-9a-f]{2}:){5}[0-9a-f]{2}$/i.test(value.trim()) ? '' : 'Enter six hexadecimal pairs separated by colons, for example 02:00:00:00:00:01.'
}
export function nameError(value: string): string {
  return /^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$/.test(value) ? '' : 'Use 1–63 letters, digits, dots, underscores or hyphens; begin with a letter or digit.'
}
export function hostnameError(value: string): string {
  return !value.trim() || /^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(\.([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*$/.test(value.trim()) ? '' : 'Enter a valid hostname, for example web-01 or db.internal.'
}
export function ipError(value: string): string {
  if (!value.trim()) return ''
  const address = value.trim()
  const v4 = address.split('.')
  const isV4 = v4.length === 4 && v4.every(part => /^\d{1,3}$/.test(part) && Number(part) <= 255)
  const halves = address.split('::')
  const groups = address.split(':').filter(Boolean)
  const isV6 = address.includes(':') && halves.length <= 2 && groups.every(part => /^[0-9a-f]{1,4}$/i.test(part)) && (halves.length === 2 ? groups.length < 8 : groups.length === 8)
  return isV4 || isV6 ? '' : 'Enter an IP address, for example 192.168.1.1 or 2001:db8::1.'
}
export function splitList(value: string): string[] {
  return value.split(/[\s,]+/).map(item => item.trim()).filter(Boolean)
}
export function dnsError(value: string): string {
  return splitList(value).some(item => !!ipError(item)) ? 'Enter IP addresses separated by commas, for example 1.1.1.1, 8.8.8.8.' : ''
}

export function safeReturnPath(value: string | null): string {
  return value && value.startsWith('/') && !value.startsWith('//') && !value.includes('\\') && !value.startsWith('/login') ? value : '/'
}

export interface NavOrigin {
  pathname: string
  search: string
}

export function navOrigin(location: { pathname: string; search: string }): NavOrigin {
  return { pathname: location.pathname, search: location.search }
}

export function pageLabel(pathname: string): string {
  if (pathname === '/') return 'Virtual Machines'
  if (pathname.startsWith('/vms/')) return 'the virtual machine'
  if (pathname.startsWith('/networks')) return 'Networks'
  if (pathname.startsWith('/disks')) return 'Disks'
  if (pathname.startsWith('/media')) return 'Images & ISOs'
  if (pathname.startsWith('/jobs')) return 'Activity'
  if (pathname.startsWith('/api-docs')) return 'API'
  if (pathname.startsWith('/console')) return 'Console'
  if (pathname.startsWith('/profile')) return 'Profile'
  return 'previous page'
}
