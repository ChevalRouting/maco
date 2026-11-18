import { type CatalogImage, type CreateVMParams, type GuestCapability, type Media, type Network } from '../api'
import { macError } from '../ux'
import { type GuestSetupValue } from './GuestSetup'
import { vmNetworkChoices } from './VMNetwork'

export type SpecSource = 'image' | 'iso'

export interface DiskDraft {
  name: string
  size_gib: number
}

export interface AdapterDraft {
  network: string
  mac: string
}

export interface SpecDraft {
  source: SpecSource
  image: string
  iso: string
  cpus: number
  memory_mib: number
  disk_size_gib: number
  autostart: boolean
  tags: string[]
  disks: DiskDraft[]
  interfaces: AdapterDraft[]
  addresses: string[]
  guestEnabled: boolean
  guest: GuestSetupValue
}

export interface SpecResources {
  catalog: CatalogImage[]
  media: Media[]
  networks: Network[]
  networksLoading: boolean
}

export const defaultAdapter: AdapterDraft = { network: 'user', mac: '' }

const defaultImage = 'ubuntu-24.04-arm64'
const cloudInit: GuestCapability[] = [{ provisioner: 'cloud-init', raw: true }]

function guestFromSpec(spec: CreateVMParams): GuestSetupValue {
  const setup = spec.guest_setup
  const primary = setup?.provisioner === 'ignition' ? 'config.bu' : 'user-data'
  const files = setup?.files?.length
    ? setup.files.map((file) => ({ name: file.name ?? '', content: file.content ?? '' }))
    : setup?.raw
      ? [{ name: primary, content: setup.raw }]
      : []
  const enabled = !!setup?.provisioner && setup.provisioner !== 'none'
  return { provisioner: enabled ? setup!.provisioner! : '', files }
}

export function specToDraft(spec?: CreateVMParams): SpecDraft {
  const iso = spec?.isos?.[0] ?? ''
  const provisioner = spec?.guest_setup?.provisioner
  return {
    source: iso ? 'iso' : 'image',
    image: spec?.image || defaultImage,
    iso,
    cpus: spec?.cpus || 2,
    memory_mib: spec?.memory_mib || 2048,
    disk_size_gib: spec?.disk_size_gib || 20,
    autostart: spec?.autostart ?? false,
    tags: spec?.tags ?? [],
    disks: (spec?.disks ?? []).map((disk) => ({ name: disk.name, size_gib: disk.size_gib })),
    interfaces: spec?.interfaces?.length
      ? spec.interfaces.map((nic) => ({ network: nic.network || 'user', mac: nic.mac || '' }))
      : [{ network: spec?.network || 'user', mac: '' }],
    addresses: spec?.addresses ?? [],
    guestEnabled: !!provisioner && provisioner !== 'none',
    guest: spec ? guestFromSpec(spec) : { provisioner: '', files: [] },
  }
}

export function imageSize(draft: SpecDraft, resources: SpecResources): number {
  if (draft.source !== 'image') return 1
  return resources.media.find((item) => `media:${item.id}` === draft.image)?.size_gib || 1
}

export function guestCapabilities(draft: SpecDraft, resources: SpecResources): GuestCapability[] {
  if (draft.image.startsWith('media:')) return cloudInit
  return resources.catalog.find((item) => item.id === draft.image)?.provisioning || cloudInit
}

export function draftToSpec(draft: SpecDraft, resources: SpecResources, name: string): CreateVMParams {
  const iso = draft.source === 'iso'
  const guest = !iso && draft.guestEnabled
  return {
    name,
    image: iso ? '' : draft.image,
    isos: iso ? [draft.iso] : [],
    boot_order: iso ? [`iso:${draft.iso}`, 'disk'] : ['disk'],
    cpus: draft.cpus,
    memory_mib: draft.memory_mib,
    disk_size_gib: Math.max(draft.disk_size_gib, imageSize(draft, resources)),
    autostart: draft.autostart,
    tags: draft.tags,
    addresses: draft.addresses,
    disks: draft.disks.map((disk) => ({ name: disk.name.trim(), size_gib: disk.size_gib })),
    interfaces: draft.interfaces.map((nic) => ({ network: nic.network, mac: nic.mac.trim() })),
    guest_setup: guest
      ? { provisioner: draft.guest.provisioner, mode: 'raw', files: draft.guest.files }
      : iso
        ? undefined
        : { provisioner: 'none' },
  }
}

export function sourceAvailable(draft: SpecDraft, resources: SpecResources): boolean {
  if (draft.source === 'iso') return resources.media.some((item) => item.kind === 'iso' && item.id === draft.iso)
  return (
    resources.catalog.some((item) => item.id === draft.image) ||
    resources.media.some((item) => item.kind === 'image' && `media:${item.id}` === draft.image)
  )
}

export function specValid(draft: SpecDraft, resources: SpecResources): boolean {
  const choices = vmNetworkChoices(resources.networks)
  const adaptersValid = draft.interfaces.every(
    (nic) => choices.some((item) => item.value === nic.network && !item.disabled) && !macError(nic.mac),
  )
  const disksValid = draft.disks.every((disk) => disk.name.trim() && disk.size_gib >= 1)
  const guestValid = draft.source === 'iso' || !draft.guestEnabled || draft.guest.files.some((file) => file.content.trim())
  return sourceAvailable(draft, resources) && adaptersValid && disksValid && guestValid
}
