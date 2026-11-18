import { useState } from 'react'
import { EntryRow, NoticeBanner, PreferencesGroup, Row, Segmented, SwitchRow, TagInput } from 'maco-ui'
import { Plus, Shuffle, Trash2 } from 'lucide-react'
import { suggestMAC, type HostInfo } from '../api'
import { macError, memoryLabel } from '../ux'
import { ActionButton } from './ActionButton'
import { ChoiceRow } from './ChoiceRow'
import { GuestSetup } from './GuestSetup'
import { IntegerEntryRow } from './IntegerEntryRow'
import { MemoryEntryRow } from './MemoryEntryRow'
import { vmNetworkChoices } from './VMNetwork'
import {
  defaultAdapter,
  guestCapabilities,
  imageSize,
  type AdapterDraft,
  type DiskDraft,
  type SpecDraft,
  type SpecResources,
  type SpecSource,
} from './vmSpec'

interface VMSpecFieldsProps {
  idPrefix: string
  draft: SpecDraft
  onChange: (draft: SpecDraft) => void
  resources: SpecResources
  host: HostInfo | null
  disabled?: boolean
}

interface SourceOption {
  value: SpecSource
  label: string
}

const sourceOptions: SourceOption[] = [
  { value: 'image', label: 'Disk Image' },
  { value: 'iso', label: 'ISO' },
]

export function VMSpecFields({ idPrefix, draft, onChange, resources, host, disabled }: VMSpecFieldsProps) {
  const [activeFile, setActiveFile] = useState('')
  const minDisk = imageSize(draft, resources)
  const isos = resources.media.filter((item) => item.kind === 'iso')
  const imageChoices = [
    ...resources.catalog.map((image) => ({ value: image.id, label: `${image.display_name} · ${image.arch}` })),
    ...resources.media
      .filter((item) => item.kind === 'image')
      .map((item) => ({ value: `media:${item.id}`, label: `${item.name} · ${item.size_gib} GiB` })),
  ]
  const selectedCatalog = resources.catalog.find((item) => item.id === draft.image)
  const overCapacity = host && (draft.cpus > host.cpus || draft.memory_mib * 1048576 > host.memory_bytes)

  function set(patch: Partial<SpecDraft>) {
    onChange({ ...draft, ...patch })
  }

  function setDisk(index: number, patch: Partial<DiskDraft>) {
    set({ disks: draft.disks.map((disk, i) => (i === index ? { ...disk, ...patch } : disk)) })
  }

  function setAdapter(index: number, patch: Partial<AdapterDraft>) {
    set({ interfaces: draft.interfaces.map((nic, i) => (i === index ? { ...nic, ...patch } : nic)) })
  }

  async function suggest(index: number) {
    const suggestion = await suggestMAC()
    setAdapter(index, { mac: suggestion.mac })
  }

  return (
    <>
      <PreferencesGroup title="Source">
        <Row title="Install From">
          <Segmented value={draft.source} onChange={(source) => set({ source })} options={sourceOptions} />
        </Row>
        {draft.source === 'image' ? (
          <ChoiceRow
            title="Image"
            value={draft.image}
            choices={imageChoices}
            disabled={disabled}
            help={selectedCatalog && !selectedCatalog.downloaded ? 'Downloads on first use' : undefined}
            onChange={(image) => set({ image })}
          />
        ) : (
          <ChoiceRow
            title="ISO"
            value={draft.iso}
            choices={isos.map((item) => ({ value: item.id, label: item.name }))}
            disabled={disabled}
            onChange={(iso) => set({ iso })}
          />
        )}
      </PreferencesGroup>

      <PreferencesGroup title="Hardware">
        <IntegerEntryRow
          id={`${idPrefix}-cpus`}
          title="vCPUs"
          min={1}
          required
          value={draft.cpus}
          help={host ? `Host has ${host.cpus}` : undefined}
          onValueChange={(cpus) => set({ cpus })}
        />
        <MemoryEntryRow
          id={`${idPrefix}-memory`}
          value={draft.memory_mib}
          help={host ? `Host has ${memoryLabel(host.memory_bytes / 1048576)}` : undefined}
          onValueChange={(memory_mib) => set({ memory_mib })}
        />
        <IntegerEntryRow
          id={`${idPrefix}-disk`}
          title="Boot Disk (GiB)"
          min={minDisk}
          required
          value={Math.max(draft.disk_size_gib, minDisk)}
          onValueChange={(disk_size_gib) => set({ disk_size_gib })}
        />
        <SwitchRow
          title="Start Automatically"
          subtitle="Started again at each reconcile, even after a manual shutdown"
          checked={draft.autostart}
          onCheckedChange={(autostart) => set({ autostart })}
        />
      </PreferencesGroup>
      {overCapacity && <NoticeBanner intent="warning">Exceeds the host’s CPUs or memory</NoticeBanner>}

      <PreferencesGroup title="Tags">
        <div className="px-4 py-3">
          <TagInput values={draft.tags} placeholder="Add a tag" onChange={(tags) => set({ tags })} />
        </div>
      </PreferencesGroup>

      <PreferencesGroup
        title="Additional Disks"
        header={
          <ActionButton label="Add Disk" icon={Plus} disabled={disabled} onClick={() => set({ disks: [...draft.disks, { name: '', size_gib: 10 }] })} />
        }
      >
        {draft.disks.length === 0 && <Row subtitle="None" />}
        {draft.disks.map((disk, index) => (
          <div key={index} className="flex items-center">
            <div className="min-w-0 flex-1 divide-y divide-border">
              <EntryRow
                id={`${idPrefix}-disk-name-${index}`}
                title={`Disk ${index + 2}`}
                placeholder="data"
                required
                value={disk.name}
                onChange={(event) => setDisk(index, { name: event.target.value })}
              />
              <IntegerEntryRow
                id={`${idPrefix}-disk-size-${index}`}
                title="Capacity (GiB)"
                min={1}
                required
                value={disk.size_gib}
                onValueChange={(size_gib) => setDisk(index, { size_gib })}
              />
            </div>
            <div className="px-2">
              <ActionButton label={`Remove Disk ${index + 2}`} icon={Trash2} onClick={() => set({ disks: draft.disks.filter((_, i) => i !== index) })} />
            </div>
          </div>
        ))}
      </PreferencesGroup>

      <PreferencesGroup
        title="Network Adapters"
        header={
          <ActionButton
            label="Add Adapter"
            icon={Plus}
            disabled={disabled}
            onClick={() => set({ interfaces: [...draft.interfaces, defaultAdapter] })}
          />
        }
      >
        {draft.interfaces.length === 0 && <Row subtitle="None" />}
        {draft.interfaces.map((nic, index) => (
          <div key={index} className="flex items-center">
            <div className="min-w-0 flex-1 divide-y divide-border">
              <ChoiceRow
                title={`Adapter ${index + 1}`}
                value={nic.network}
                choices={vmNetworkChoices(resources.networks, nic.network, resources.networksLoading)}
                disabled={disabled || resources.networksLoading}
                onChange={(network) => setAdapter(index, { network })}
              />
              <EntryRow
                id={`${idPrefix}-mac-${index}`}
                title="MAC Address"
                placeholder="Automatic"
                className="font-mono"
                value={nic.mac}
                error={macError(nic.mac)}
                onChange={(event) => setAdapter(index, { mac: event.target.value })}
                suffix={<ActionButton label="Suggest MAC Address" icon={Shuffle} onClick={() => void suggest(index)} />}
              />
            </div>
            <div className="px-2">
              <ActionButton
                label={`Remove Adapter ${index + 1}`}
                icon={Trash2}
                onClick={() => set({ interfaces: draft.interfaces.filter((_, i) => i !== index) })}
              />
            </div>
          </div>
        ))}
      </PreferencesGroup>

      {draft.source === 'image' && (
        <PreferencesGroup title="Guest Setup">
          <SwitchRow
            title="Provide Configuration"
            subtitle="Applied on the first boot of a fresh disk"
            checked={draft.guestEnabled}
            onCheckedChange={(guestEnabled) => set({ guestEnabled })}
          />
          {draft.guestEnabled && (
            <GuestSetup
              value={draft.guest}
              onChange={(guest) => set({ guest })}
              capabilities={guestCapabilities(draft, resources)}
              disabled={disabled}
              activeFile={activeFile}
              onActiveFileChange={setActiveFile}
            />
          )}
        </PreferencesGroup>
      )}
    </>
  )
}

