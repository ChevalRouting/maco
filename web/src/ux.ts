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
  'vm.snapshot.delete': 'Delete Snapshot', 'vm.reboot': 'Reboot Virtual Machine',
  'vm.disk.wipe': 'Wipe Disk', 'vm.disk.replace': 'Replace Boot Disk', 'vm.guest-setup': 'Save Guest Setup',
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const exp = Math.max(0, Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1))
  const value = bytes / Math.pow(1024, exp)
  return `${Number(value.toFixed(exp === 0 && bytes >= 1 ? 0 : 1))} ${units[exp]}`
}

export function activityName(job: Job): string {
  return actionNames[job.action] || job.label?.trim() || job.action || 'Job'
}
export function activityState(job: Job): string {
  if (job.action === 'vm.shutdown' && job.state === 'succeeded') return 'Shutdown Requested'
  return { pending: 'Queued', running: 'In Progress', succeeded: 'Completed', failed: 'Failed' }[job.state] || 'Unknown'
}
export function unfinishedTime(job: Job): string {
  return job.state === 'pending' ? 'Not started' : job.state === 'running' ? 'In progress' : 'Not recorded'
}
export function memoryLabel(mib: number): string {
  return `${Number((mib / 1024).toFixed(10))} GiB`
}

export const networkModes = [
  { value: 'user', label: 'User NAT', help: 'Per-adapter NAT with DHCP, no inbound connections' },
  { value: 'switch', label: 'Isolated Switch', help: 'VM to VM only, no DHCP' },
  { value: 'bridge', label: 'Bridge', help: 'Bridges host interfaces, no DHCP or NAT' },
  { value: 'vmnet-bridged', label: 'vmnet Bridged', help: 'Joins the adapter’s network, requires administrator rights' },
  { value: 'vlan', label: 'VLAN', help: 'Tagged interface to add to a host bridge' },
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
  return builtin?.help || networkModes.find(item => item.value === network?.mode)?.help || 'Network unavailable'
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
    return ipv4Only ? 'IPv4 address with prefix, for example 192.168.1.20/24' : 'Address with prefix, for example 192.168.1.20/24 or 2001:db8::20/64'
  }
  return ''
}
export function macError(value: string): string {
  return !value.trim() || /^([0-9a-f]{2}:){5}[0-9a-f]{2}$/i.test(value.trim()) ? '' : 'Six hexadecimal pairs, for example 02:00:00:00:00:01'
}
export function nameError(value: string): string {
  return /^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$/.test(value) ? '' : '1–63 letters, digits, dots, underscores or hyphens, starting with a letter or digit'
}
export function safeReturnPath(value: string | null): string {
  return value && value.startsWith('/') && !value.startsWith('//') && !value.includes('\\') && !value.startsWith('/login') ? value : '/'
}

export interface NavOrigin {
  pathname: string
  search: string
}

export function navOrigin(location: NavOrigin): NavOrigin {
  return { pathname: location.pathname, search: location.search }
}

export function pageLabel(pathname: string): string {
  if (pathname === '/') return 'Virtual Machines'
  if (pathname.startsWith('/vms/')) return 'Virtual Machine'
  if (pathname.startsWith('/networks')) return 'Networks'
  if (pathname.startsWith('/disks')) return 'Disks'
  if (pathname.startsWith('/media')) return 'Media'
  if (pathname.startsWith('/jobs')) return 'Activity'
  if (pathname.startsWith('/api-docs')) return 'API'
  if (pathname.startsWith('/console')) return 'Console'
  if (pathname.startsWith('/preferences')) return 'Preferences'
  return 'Back'
}

export function plural(count: number, noun: string): string {
  return `${count} ${count === 1 ? noun : `${noun}s`}`
}
