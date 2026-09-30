import { useMemo, useState, type FormEvent } from 'react'
import { Dialog, Button, PreferencesGroup, SwitchRow, EntryRow } from 'cheval-ui'
import {
  createTemplate,
  updateTemplate,
  listCatalog,
  listMedia,
  listNetworks,
  errorMessage,
  type CreateVMParams,
  type GuestCapability,
  type Template,
} from '../api'
import { useResource } from '../hooks/useResource'
import { ChoiceRow } from './ChoiceRow'
import { ImagePicker, type ImageOption } from './ImagePicker'
import { vmNetworkChoices } from './VMNetwork'
import { ResourceNotice } from './ResourceNotice'
import { IntegerEntryRow } from './IntegerEntryRow'
import { MemoryEntryRow } from './MemoryEntryRow'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { GuestSetup, type GuestSetupValue } from './GuestSetup'
import { TemplateVariables } from './TemplateVariables'
import { nameError } from '../ux'

const guestDefaults: GuestSetupValue = {
  provisioner: '',
  files: [],
}

const cloudInitCapabilities: GuestCapability[] = [{ provisioner: 'cloud-init', raw: true }]

interface HardwareForm {
  image: string
  cpus: number
  memory_mib: number
  disk_size_gib: number
  autostart: boolean
  network: string
}

const hardwareDefaults: HardwareForm = {
  image: 'ubuntu-24.04-arm64',
  cpus: 2,
  memory_mib: 2048,
  disk_size_gib: 20,
  autostart: false,
  network: 'user',
}

function seedFromTemplate(template?: Template) {
  const spec = template?.spec
  if (!spec) {
    return {
      name: '',
      description: '',
      hardware: hardwareDefaults,
      disks: [] as { name: string; size_gib: number }[],
      guest: guestDefaults,
      guestEnabled: false,
    }
  }
  const gs = spec.guest_setup
  const guestEnabled = !!gs && !!gs.provisioner && gs.provisioner !== 'none'
  return {
    name: template?.name ?? '',
    description: template?.description ?? '',
    hardware: {
      image: spec.image || hardwareDefaults.image,
      cpus: spec.cpus || hardwareDefaults.cpus,
      memory_mib: spec.memory_mib || hardwareDefaults.memory_mib,
      disk_size_gib: spec.disk_size_gib || hardwareDefaults.disk_size_gib,
      autostart: spec.autostart ?? false,
      network: spec.interfaces?.[0]?.network || hardwareDefaults.network,
    },
    disks: (spec.disks ?? []).map((disk) => ({ name: disk.name, size_gib: disk.size_gib })),
    guest: {
      provisioner: guestEnabled ? gs!.provisioner! : '',
      files:
        gs?.files?.map((file) => ({ name: file.name ?? '', content: file.content ?? '' })) ??
        (gs?.raw
          ? [{ name: gs.provisioner === 'ignition' ? 'config.bu' : 'user-data', content: gs.raw }]
          : []),
    } as GuestSetupValue,
    guestEnabled,
  }
}

export function TemplateEditor({ template, onClose }: { template?: Template; onClose: () => void }) {
  const seed = useMemo(() => seedFromTemplate(template), [template])
  const [name, setName] = useState(seed.name)
  const [description, setDescription] = useState(seed.description)
  const [hardware, setHardware] = useState(seed.hardware)
  const [disks, setDisks] = useState(seed.disks)
  const [guest, setGuest] = useState<GuestSetupValue>(seed.guest)
  const [guestEnabled, setGuestEnabled] = useState(seed.guestEnabled)
  const [activeFile, setActiveFile] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')
  const networks = useResource(listNetworks, [], 'networks')

  const imageOptions: ImageOption[] = [
    ...catalog.data.map((image) => ({
      value: image.id,
      distro: image.distro,
      label: `${image.display_name} · ${image.arch}`,
      sub: 'A new VM created from this template receives an independent copy.',
    })),
    ...media.data
      .filter((item) => item.kind === 'image')
      .map((item) => ({
        value: `media:${item.id}`,
        distro: '',
        label: item.name,
        sub: `Custom disk image · ${item.size_gib} GiB.`,
      })),
  ]

  const custom = hardware.image.startsWith('media:')
  const guestCapabilities = !custom
    ? catalog.data.find((item) => item.id === hardware.image)?.provisioning || cloudInitCapabilities
    : cloudInitCapabilities

  const invalidDisks = disks.some((disk) => !disk.name.trim() || disk.size_gib < 1)
  const nameProblem = name ? nameError(name) : 'A template name is required.'

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (nameProblem || invalidDisks || busy) return
    const spec: CreateVMParams = {
      name: '',
      image: hardware.image,
      cpus: hardware.cpus,
      memory_mib: hardware.memory_mib,
      disk_size_gib: hardware.disk_size_gib,
      autostart: hardware.autostart,
      disks: disks.map((disk) => ({ name: disk.name.trim(), size_gib: disk.size_gib })),
      interfaces: [{ network: hardware.network, mac: '' }],
      guest_setup: guestEnabled
        ? {
            provisioner: guest.provisioner,
            mode: 'raw',
            files: guest.files,
          }
        : { provisioner: 'none' },
    }
    setBusy(true)
    setError('')
    try {
      if (template) {
        await updateTemplate(template.id, { name, description, spec })
      } else {
        await createTemplate({ name, description, spec })
      }
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open
      onClose={() => !busy && onClose()}
      title={template ? 'Edit Template' : 'New Template'}
      description="Save a reusable machine profile: guest provisioning plus hardware defaults."
      className="max-w-2xl"
      footer={
        <>
          <Button variant="outline" onClick={() => !busy && onClose()} disabled={busy}>
            Cancel
          </Button>
          <Button
            variant="suggested"
            type="submit"
            form="template-editor"
            disabled={busy || !!nameProblem || invalidDisks}
          >
            {busy ? 'Saving…' : 'Save'}
          </Button>
        </>
      }
    >
      <form id="template-editor" onSubmit={submit} className="space-y-4">
        {error && <p className="text-sm text-destructive">{error}</p>}
        <fieldset disabled={busy} className="space-y-4">
          <PreferencesGroup title="Template">
            <ValidatedEntryRow
              id="template-name"
              title="Name"
              required
              autoFocus
              value={name}
              error={name ? nameError(name) : ''}
              help="1–63 letters, digits, dots, underscores or hyphens"
              onChange={(event) => setName(event.target.value)}
            />
            <EntryRow
              id="template-description"
              title="Description (optional)"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </PreferencesGroup>

          <PreferencesGroup title="Image">
            <ResourceNotice resource={catalog} name="system images" />
            <ResourceNotice resource={media} name="disk images" />
            {!catalog.loading && (
              <ImagePicker
                title="Base image"
                value={hardware.image}
                options={imageOptions}
                onChange={(image) => setHardware({ ...hardware, image })}
              />
            )}
          </PreferencesGroup>

          <PreferencesGroup title="Hardware">
            <IntegerEntryRow
              id="template-cpus"
              title="Virtual CPUs"
              min={1}
              required
              value={hardware.cpus}
              onValueChange={(cpus) => setHardware({ ...hardware, cpus })}
            />
            <MemoryEntryRow
              id="template-memory"
              value={hardware.memory_mib}
              onValueChange={(memory_mib) => setHardware({ ...hardware, memory_mib })}
            />
            <IntegerEntryRow
              id="template-disk"
              title="Boot disk capacity (GiB)"
              min={1}
              required
              value={hardware.disk_size_gib}
              onValueChange={(disk_size_gib) => setHardware({ ...hardware, disk_size_gib })}
            />
            <SwitchRow
              title="Automatic Startup"
              subtitle="Start VMs created from this template when the host reconciles."
              checked={hardware.autostart}
              onCheckedChange={(autostart) => setHardware({ ...hardware, autostart })}
            />
          </PreferencesGroup>

          <PreferencesGroup
            title="Additional Disks"
            description="Extra blank data disks. Partition and format them from inside the guest."
          >
            {disks.map((disk, index) => (
              <div key={index} className="flex items-start">
                <div className="min-w-0 flex-1">
                  <ValidatedEntryRow
                    id={`template-disk-name-${index}`}
                    title={`Disk ${index + 1} name`}
                    required
                    value={disk.name}
                    error={disk.name.trim() ? '' : 'Enter a disk name.'}
                    onChange={(event) =>
                      setDisks(disks.map((item, i) => (i === index ? { ...item, name: event.target.value } : item)))
                    }
                  />
                  <IntegerEntryRow
                    id={`template-disk-size-${index}`}
                    title="Capacity (GiB)"
                    min={1}
                    required
                    value={disk.size_gib}
                    onValueChange={(size_gib) =>
                      setDisks(disks.map((item, i) => (i === index ? { ...item, size_gib } : item)))
                    }
                  />
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  className="mt-3 mr-2"
                  aria-label={`Remove disk ${index + 1}`}
                  onClick={() => setDisks(disks.filter((_, i) => i !== index))}
                >
                  Remove
                </Button>
              </div>
            ))}
            <Button type="button" className="m-3" onClick={() => setDisks([...disks, { name: '', size_gib: 10 }])}>
              Add Disk
            </Button>
          </PreferencesGroup>

          <PreferencesGroup title="Network">
            <ResourceNotice resource={networks} name="networks" />
            <ChoiceRow
              stacked
              title="Default network"
              value={hardware.network}
              choices={vmNetworkChoices(networks.data, hardware.network, networks.loading)}
              disabled={networks.loading}
              onChange={(network) => setHardware({ ...hardware, network })}
            />
          </PreferencesGroup>

          <PreferencesGroup>
            <SwitchRow
              title="Enable guest configuration"
              subtitle="Author cloud-init or Ignition here. When off, VMs boot with the image's built-in configuration."
              checked={guestEnabled}
              onCheckedChange={setGuestEnabled}
            />
          </PreferencesGroup>
          {guestEnabled && (
            <>
              <TemplateVariables />
              <GuestSetup
                value={guest}
                onChange={setGuest}
                capabilities={guestCapabilities}
                disabled={busy}
                activeFile={activeFile}
                onActiveFileChange={setActiveFile}
              />
            </>
          )}
        </fieldset>
      </form>
    </Dialog>
  )
}
