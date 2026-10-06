import { useMemo, useState, type FormEvent } from 'react'
import { Dialog, Button, PreferencesGroup, NoticeBanner, SwitchRow, TagInput } from 'cheval-ui'
import { createVM, listCatalog, listMedia, listNetworks, getHost, suggestMAC, type CreateVMParams, type GuestCapability } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { ChoiceRow } from './ChoiceRow'
import { ImagePicker, type ImageOption } from './ImagePicker'
import { vmNetworkChoices } from './VMNetwork'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { IntegerEntryRow } from './IntegerEntryRow'
import { MemoryEntryRow } from './MemoryEntryRow'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { GuestSetup, type GuestSetupValue } from './GuestSetup'
import { TemplateVariables } from './TemplateVariables'
import { automaticStartupHelp, networkHelp, memoryLabel, nameError, macError } from '../ux'

type VMForm = Required<Omit<CreateVMParams, 'isos' | 'boot_order' | 'guest_setup' | 'gateway' | 'nameservers' | 'disks' | 'interfaces' | 'network'>> & Pick<CreateVMParams, 'isos' | 'boot_order'>

const defaults: VMForm = {
  name: '',
  image: 'ubuntu-24.04-arm64',
  cpus: 2,
  memory_mib: 2048,
  disk_size_gib: 20,
  addresses: [],
  tags: [],
  autostart: false,
}

const guestDefaults: GuestSetupValue = {
  provisioner: '',
  files: [],
}

const cloudInitCapabilities: GuestCapability[] = [{ provisioner: 'cloud-init', raw: true }]

export interface CreateVMInitial {
  name?: string
  spec?: CreateVMParams
}

interface SeededState {
  form: VMForm
  guest: GuestSetupValue
  source: 'image' | 'iso'
  disks: { name: string; size_gib: number }[]
  interfaces: { network: string; mac: string }[]
  guestEnabled: boolean
}

function specToState(initial?: CreateVMInitial): SeededState {
  const spec = initial?.spec
  if (!spec) {
    return {
      form: { ...defaults, name: initial?.name ?? '' },
      guest: guestDefaults,
      source: 'image',
      disks: [],
      interfaces: [{ network: 'user', mac: '' }],
      guestEnabled: false,
    }
  }
  const source: 'image' | 'iso' = spec.isos && spec.isos.length ? 'iso' : 'image'
  const gs = spec.guest_setup
  const guestEnabled = !!gs && !!gs.provisioner && gs.provisioner !== 'none'
  const form: VMForm = {
    name: initial?.name ?? '',
    image: source === 'image' ? spec.image || defaults.image : defaults.image,
    cpus: spec.cpus || defaults.cpus,
    memory_mib: spec.memory_mib || defaults.memory_mib,
    disk_size_gib: spec.disk_size_gib || defaults.disk_size_gib,
    addresses: spec.addresses ?? [],
    tags: spec.tags ?? [],
    autostart: spec.autostart ?? false,
    isos: spec.isos ?? [],
  }
  const primaryFile = gs?.provisioner === 'ignition' ? 'config.bu' : 'user-data'
  const files =
    gs?.files?.length
      ? gs.files.map((file) => ({ name: file.name ?? '', content: file.content ?? '' }))
      : gs?.raw
        ? [{ name: primaryFile, content: gs.raw }]
        : []
  const guest: GuestSetupValue = {
    provisioner: guestEnabled ? gs!.provisioner! : '',
    files,
  }
  const disks = (spec.disks ?? []).map((disk) => ({ name: disk.name, size_gib: disk.size_gib }))
  const interfaces =
    spec.interfaces && spec.interfaces.length
      ? spec.interfaces.map((nic) => ({ network: nic.network || 'user', mac: nic.mac || '' }))
      : [{ network: 'user', mac: '' }]
  return { form, guest, source, disks, interfaces, guestEnabled }
}

export function CreateVMDialog({
  onClose,
  initial,
}: {
  onClose: () => void
  initial?: CreateVMInitial
}) {
  const seed = useMemo(() => specToState(initial), [])
  const [form, setForm] = useState(seed.form)
  const [guest, setGuest] = useState<GuestSetupValue>(seed.guest)
  const [source, setSource] = useState<'image' | 'iso'>(seed.source)
  const [disks, setDisks] = useState<{ name: string; size_gib: number }[]>(seed.disks)
  const [interfaces, setInterfaces] = useState<{ network: string; mac: string }[]>(seed.interfaces)
  const [guestEnabled, setGuestEnabled] = useState(seed.guestEnabled)
  const [activeFile, setActiveFile] = useState('')
  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')
  const networks = useResource(listNetworks, [], 'networks')
  const host = useResource(getHost, null, 'host')
  const action = useJobAction()

  const imageOptions: ImageOption[] = [
    ...catalog.data.map((image) => ({
      value: image.id,
      distro: image.distro,
      label: `${image.display_name} · ${image.arch}`,
      sub: image.downloaded
        ? 'Downloaded. A new VM receives an independent copy.'
        : 'Downloads on first use. Creation continues in Activity.',
    })),
    ...media.data
      .filter((item) => item.kind === 'image')
      .map((item) => ({
        value: `media:${item.id}`,
        distro: '',
        label: item.name,
        sub: `Custom disk image · ${item.size_gib} GiB. A new VM receives an independent copy.`,
      })),
  ]
  const isos = media.data.filter((item) => item.kind === 'iso')

  const selectedImage = form.image
  const selectedISO = form.isos?.[0] || ''
  const ready = !media.loading && (source === 'iso' || !catalog.loading) && !networks.loading
  const sourceError = source === 'image' ? catalog.error || media.error : media.error
  const sourceAvailable =
    source === 'image'
      ? imageOptions.some((item) => item.value === selectedImage)
      : isos.some((item) => item.id === selectedISO)
  const custom = source === 'image' && selectedImage.startsWith('media:')
  const guestCapabilities =
    source === 'image' && !custom
      ? catalog.data.find((item) => item.id === selectedImage)?.provisioning || cloudInitCapabilities
      : cloudInitCapabilities
  const applyGuest = source === 'image' && guestEnabled
  const guestRawMissing = applyGuest && !guest.files.some((file) => file.content.trim())
  const imageSize =
    source === 'image'
      ? media.data.find((item) => `media:${item.id}` === selectedImage)?.size_gib || 1
      : 1
  const diskSize = Math.max(form.disk_size_gib, imageSize)
  const networkChoices = vmNetworkChoices(networks.data)
  const validNetwork = interfaces.every(
    (nic) =>
      networkChoices.some((item) => item.value === nic.network && !item.disabled) &&
      !macError(nic.mac),
  )
  const invalidDisks = disks.some((disk) => !disk.name.trim() || disk.size_gib < 1)
  const blocked = action.busy

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (
      !ready ||
      !sourceAvailable ||
      !validNetwork ||
      sourceError ||
      networks.error ||
      invalidDisks ||
      guestRawMissing ||
      nameError(form.name)
    ) {
      return
    }
    const job = await action.run(() =>
      createVM({
        ...form,
        image: source === 'image' ? selectedImage : '',
        isos: source === 'iso' ? [selectedISO] : [],
        boot_order: source === 'iso' ? [`iso:${selectedISO}`, 'disk'] : ['disk'],
        disk_size_gib: diskSize,
        disks: disks.map((disk) => ({ name: disk.name.trim(), size_gib: disk.size_gib })),
        interfaces: interfaces.map((nic) => ({ network: nic.network, mac: nic.mac.trim() })),
        guest_setup: applyGuest
          ? {
              provisioner: guest.provisioner,
              mode: 'raw',
              files: guest.files,
            }
          : source === 'image'
            ? { provisioner: 'none' }
            : undefined,
      }),
    )
    if (job) onClose()
  }

  function close() {
    if (!blocked) onClose()
  }

  async function suggestNicMAC(index: number) {
    const suggestion = await suggestMAC()
    setInterfaces((current) =>
      current.map((nic, i) => (i === index ? { ...nic, mac: suggestion.mac } : nic)),
    )
  }

  const overCapacity =
    host.data && (form.cpus > host.data.cpus || form.memory_mib * 1048576 > host.data.memory_bytes)

  return (
    <Dialog
      open
      onClose={close}
      title="Create Virtual Machine"
      description="Choose an installation source and guest access. Hardware defaults can be adjusted below."
      className="max-w-2xl"
      footer={
        <>
          <Button variant="outline" onClick={close} disabled={blocked}>
            Cancel
          </Button>
          <Button
            variant="suggested"
            type="submit"
            form="create-vm"
            disabled={
              blocked ||
              !ready ||
              !sourceAvailable ||
              !!sourceError ||
              !!networks.error ||
              !validNetwork ||
              guestRawMissing ||
              invalidDisks
            }
          >
            {action.busy ? 'Submitting…' : 'Create'}
          </Button>
        </>
      }
    >
      <form id="create-vm" onSubmit={submit} className="space-y-4">
        <JobNotice error={action.error} />

        <fieldset disabled={blocked} className="space-y-4">
          <PreferencesGroup title="Virtual Machine">
            <ValidatedEntryRow
              id="vm-name"
              title="Name"
              required
              autoFocus
              value={form.name}
              error={form.name ? nameError(form.name) : ''}
              help="1–63 letters, digits, dots, underscores or hyphens"
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
            <ChoiceRow
              stacked
              title="Install from"
              value={source}
              choices={[
                { value: 'image', label: 'Disk image' },
                { value: 'iso', label: 'Installation ISO' },
              ]}
              onChange={(value) => setSource(value as 'image' | 'iso')}
            />
            <ResourceNotice resource={media} name="installation media" />
            {source === 'image' && <ResourceNotice resource={catalog} name="system images" />}

            {ready && !sourceError && (
              source === 'image' ? (
                <ImagePicker
                  title="Image"
                  value={selectedImage}
                  options={imageOptions}
                  disabled={blocked}
                  onChange={(image) => setForm({ ...form, image })}
                />
              ) : (
                <ChoiceRow
                  stacked
                  title="Installation ISO"
                  value={selectedISO}
                  choices={isos.map((item) => ({ value: item.id, label: item.name }))}
                  onChange={(id) => setForm({ ...form, isos: [id] })}
                  help="Install the operating system and create its account using the guest installer."
                />
              )
            )}

            {ready && !sourceError && !sourceAvailable && (
              <p role="status" className="px-4 py-2 text-sm">
                {selectedImage && source === 'image'
                  ? 'The selected image is unavailable. Choose another, or add one from Images & ISOs.'
                  : 'No installation media available. Add one from Images & ISOs.'}
              </p>
            )}
          </PreferencesGroup>

          <details className="rounded-xl border p-4">
            <summary className="cursor-pointer font-semibold">Hardware</summary>
            <div className="mt-3 space-y-3">
            <PreferencesGroup
              description="Assigned resources are shared with this Mac and other VMs."
            >
              <IntegerEntryRow
                id="vm-cpus"
                title="Virtual CPUs"
                min={1}
                required
                value={form.cpus}
                onValueChange={(cpus) => setForm({ ...form, cpus })}
              />
              <MemoryEntryRow
                id="vm-memory"
                value={form.memory_mib}
                onValueChange={(memory_mib) => setForm({ ...form, memory_mib })}
              />
              <IntegerEntryRow
                id="vm-disk"
                title="Boot disk capacity (GiB)"
                min={imageSize}
                required
                value={diskSize}
                help={
                  imageSize > 1
                    ? `The image needs at least ${imageSize} GiB.`
                    : 'Virtual capacity; actual host disk use grows as data is written.'
                }
                onValueChange={(disk_size_gib) => setForm({ ...form, disk_size_gib })}
              />
              <SwitchRow
                title="Automatic Startup"
                subtitle={automaticStartupHelp}
                checked={form.autostart}
                onCheckedChange={(autostart) => setForm({ ...form, autostart })}
              />
            </PreferencesGroup>
            <PreferencesGroup
              title="Tags"
              description="Freeform labels for grouping and filtering, such as role or environment."
            >
              <div className="px-4 py-3">
                <TagInput
                  values={form.tags}
                  placeholder="Add a tag"
                  onChange={(tags) => setForm({ ...form, tags })}
                />
              </div>
            </PreferencesGroup>
            <PreferencesGroup
              title="Additional Disks"
              description="Extra blank data disks attached to the VM. Partition and format them from inside the guest."
            >
              {disks.map((disk, index) => (
                <div key={index} className="flex items-start">
                  <div className="min-w-0 flex-1">
                    <ValidatedEntryRow
                      id={`vm-disk-name-${index}`}
                      title={`Disk ${index + 1} name`}
                      required
                      value={disk.name}
                      error={disk.name.trim() ? '' : 'Enter a disk name.'}
                      onChange={(event) =>
                        setDisks(
                          disks.map((item, i) =>
                            i === index ? { ...item, name: event.target.value } : item,
                          ),
                        )
                      }
                    />
                    <IntegerEntryRow
                      id={`vm-disk-size-${index}`}
                      title="Capacity (GiB)"
                      min={1}
                      required
                      value={disk.size_gib}
                      onValueChange={(size_gib) =>
                        setDisks(
                          disks.map((item, i) => (i === index ? { ...item, size_gib } : item)),
                        )
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
              <Button
                type="button"
                className="m-3"
                onClick={() => setDisks([...disks, { name: '', size_gib: 10 }])}
              >
                Add Disk
              </Button>
            </PreferencesGroup>
            <ResourceNotice resource={host} name="host capacity" />
            {host.data && (
              <p className="text-sm text-muted-foreground">
                Host capacity: {host.data.cpus} CPUs · {memoryLabel(host.data.memory_bytes / 1048576)}{' '}
                memory.
              </p>
            )}
            {overCapacity && (
              <NoticeBanner intent="warning">
                The requested assignment exceeds host capacity. Reduce it or consider other running
                workloads.
              </NoticeBanner>
            )}
            </div>
          </details>

          <details className="rounded-xl border p-4">
            <summary className="cursor-pointer font-semibold">Connectivity</summary>
            <div className="mt-3 space-y-3">
              <ResourceNotice resource={networks} name="networks" />
              {interfaces.map((nic, index) => (
                <PreferencesGroup key={index} title={`Adapter ${index + 1}`}>
                  <ChoiceRow
                    stacked
                    title="Network"
                    value={nic.network}
                    choices={vmNetworkChoices(networks.data, nic.network, networks.loading)}
                    disabled={networks.loading}
                    onChange={(network) =>
                      setInterfaces(interfaces.map((item, i) => (i === index ? { ...item, network } : item)))
                    }
                    help={networkHelp(nic.network, networks.data)}
                  />
                  <ValidatedEntryRow
                    id={`vm-nic-mac-${index}`}
                    title="MAC Address"
                    placeholder="Assigned Automatically"
                    value={nic.mac}
                    error={macError(nic.mac)}
                    help="Leave empty for automatic assignment."
                    onChange={(event) =>
                      setInterfaces(
                        interfaces.map((item, i) => (i === index ? { ...item, mac: event.target.value } : item)),
                      )
                    }
                  />
                  <div className="flex gap-2 px-4 pb-3">
                    <Button type="button" size="sm" variant="outline" onClick={() => void suggestNicMAC(index)}>
                      Suggest MAC
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => setInterfaces(interfaces.filter((_, i) => i !== index))}
                    >
                      Remove Adapter
                    </Button>
                  </div>
                </PreferencesGroup>
              ))}
              <Button
                type="button"
                onClick={() => setInterfaces([...interfaces, { network: 'user', mac: '' }])}
              >
                Add Adapter
              </Button>
            </div>
          </details>

          <details className="rounded-xl border p-4">
            <summary className="cursor-pointer font-semibold">Guest Configuration</summary>
            <div className="mt-3 space-y-3">
              {source === 'image' ? (
                <>
                  <PreferencesGroup>
                    <SwitchRow
                      title="Enable guest configuration"
                      subtitle="Deliver a cloud-init or Ignition config on first boot. When off, nothing is injected into the guest and the image boots with its own built-in configuration."
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
                        disabled={blocked}
                        activeFile={activeFile}
                        onActiveFileChange={setActiveFile}
                      />
                    </>
                  )}
                </>
              ) : (
                <p className="text-sm text-muted-foreground">
                  The installation ISO handles account creation through its own installer.
                </p>
              )}
            </div>
          </details>

        </fieldset>
      </form>
    </Dialog>
  )
}
