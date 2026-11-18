import type { VMManifest } from './api'

export function bootOrder(manifest: VMManifest, isos = manifest.isos || []): string[] {
  const hasBoot = (manifest.disk_size_gib || 0) > 0
  const devices = [
    ...(hasBoot ? ['disk'] : []),
    ...(manifest.disks || []).map(disk => `disk:${disk.id}`),
    ...isos.map(id => `iso:${id}`),
  ]
  const preferred = manifest.boot_order?.length
    ? manifest.boot_order
    : [...isos.map(id => `iso:${id}`), ...(hasBoot ? ['disk'] : [])]
  return [...new Set([...preferred, ...devices])].filter(id => devices.includes(id))
}
