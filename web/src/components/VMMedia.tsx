import { useState } from 'react'
import { NoticeBanner, PreferencesGroup, Row, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, SwitchRow } from 'maco-ui'
import { ArrowDown, ArrowUp, Disc3, HardDrive, Plus, Trash2 } from 'lucide-react'
import { listMedia, updateMedia, updateHardware, errorMessage, type VMView } from '../api'
import { bootOrder } from '../bootOrder'
import { useDraft } from '../hooks/useDraft'
import { useResource } from '../hooks/useResource'
import { ActionButton } from './ActionButton'
import { FormActions } from './FormActions'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'

interface BootDevice {
  id: string
  name: string
  detail: string
  iso: boolean
}

interface VMMediaProps {
  vm: VMView
  onSaved: () => void
}

export function VMMedia({ vm, onSaved }: VMMediaProps) {
  const draft = useDraft(`maco:draft:${vm.manifest.id}:media`, {
    isos: vm.manifest.isos || [],
    order: bootOrder(vm.manifest),
    autostart: !!vm.manifest.autostart,
  })
  const media = useResource(listMedia, [], 'media')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const available = media.data.filter((item) => item.kind === 'iso')
  const { isos, autostart } = draft.value
  const order = bootOrder({ ...vm.manifest, boot_order: draft.value.order }, isos)
  const running = vm.phase === 'running'
  const blocked = running || busy
  const missing = !media.loading && isos.some((id) => !available.some((item) => item.id === id))
  const spare = available.find((item) => !isos.includes(item.id))

  function isoName(id: string): string {
    return available.find((item) => item.id === id)?.name ?? (media.loading ? 'Loading…' : 'Missing ISO')
  }

  const devices: BootDevice[] = [
    { id: 'disk', name: 'Disk 1', detail: `${vm.manifest.disk_size_gib} GiB`, iso: false },
    ...(vm.manifest.disks || []).map((disk) => ({ id: `disk:${disk.id}`, name: disk.name, detail: `${disk.size_gib} GiB`, iso: false })),
    ...isos.map((id) => ({ id: `iso:${id}`, name: isoName(id), detail: 'CD-ROM', iso: true })),
  ]

  function addIso() {
    if (!spare) return
    draft.set({ ...draft.value, isos: [...isos, spare.id], order: [...order, `iso:${spare.id}`] })
  }

  function removeIso(id: string) {
    draft.set({ ...draft.value, isos: isos.filter((item) => item !== id), order: order.filter((item) => item !== `iso:${id}`) })
  }

  function changeIso(oldId: string, newId: string) {
    if (oldId === newId) return
    draft.set({
      ...draft.value,
      isos: isos.map((item) => (item === oldId ? newId : item)),
      order: order.map((item) => (item === `iso:${oldId}` ? `iso:${newId}` : item)),
    })
  }

  function move(index: number, delta: number) {
    const next = [...order]
    ;[next[index], next[index + delta]] = [next[index + delta], next[index]]
    draft.set({ ...draft.value, order: next })
  }

  async function save() {
    if (blocked || !draft.dirty || draft.conflict) return
    setBusy(true)
    setError('')
    try {
      const mediaChanged =
        JSON.stringify(isos) !== JSON.stringify(vm.manifest.isos || []) || JSON.stringify(order) !== JSON.stringify(bootOrder(vm.manifest))
      if (mediaChanged) await updateMedia(vm.manifest.id, isos, order)
      if (autostart !== !!vm.manifest.autostart) await updateHardware(vm.manifest.id, { autostart })
      onSaved()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-3xl space-y-6">
      <JobNotice error={error} />
      <ResourceNotice resource={media} name="ISOs" />
      {running && <NoticeBanner intent="info">Shut down to change CD-ROMs, boot order and startup</NoticeBanner>}
      {draft.conflict && <NoticeBanner intent="warning">Changed elsewhere; discard to load the saved values</NoticeBanner>}
      {missing && <NoticeBanner intent="warning">An ISO is no longer in the library</NoticeBanner>}

      <PreferencesGroup
        title="CD-ROM Drives"
        header={<ActionButton label="Add CD-ROM" icon={Plus} disabled={blocked || !spare} onClick={addIso} />}
      >
        {isos.length === 0 && <Row subtitle="None" />}
        {isos.map((id, index) => (
          <Row key={`${id}:${index}`} prefix={<Disc3 className="h-5 w-5" />} title={`CD-ROM ${index + 1}`}>
            <Select value={available.some((item) => item.id === id) ? id : undefined} disabled={blocked} onValueChange={(next) => changeIso(id, next)}>
              <SelectTrigger aria-label={`CD-ROM ${index + 1} ISO`} className="w-56 max-w-full">
                <SelectValue placeholder={isoName(id)} />
              </SelectTrigger>
              <SelectContent className="max-w-[calc(100vw-2rem)]">
                {available
                  .filter((item) => item.id === id || !isos.includes(item.id))
                  .map((item) => (
                    <SelectItem key={item.id} value={item.id} className="whitespace-normal break-words">
                      {item.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
            <ActionButton label={`Remove CD-ROM ${index + 1}`} icon={Trash2} disabled={blocked} onClick={() => removeIso(id)} />
          </Row>
        ))}
      </PreferencesGroup>

      <PreferencesGroup title="Boot Order">
        {order.map((id, index) => {
          const device = devices.find((item) => item.id === id)
          if (!device) return null
          const Icon = device.iso ? Disc3 : HardDrive
          return (
            <Row key={id} prefix={<Icon className="h-5 w-5" />} title={`${index + 1}. ${device.name}`} subtitle={device.detail}>
              <ActionButton label={`Move ${device.name} Up`} icon={ArrowUp} disabled={blocked || index === 0} onClick={() => move(index, -1)} />
              <ActionButton label={`Move ${device.name} Down`} icon={ArrowDown} disabled={blocked || index === order.length - 1} onClick={() => move(index, 1)} />
            </Row>
          )
        })}
      </PreferencesGroup>

      <PreferencesGroup title="Startup">
        <SwitchRow
          title="Start Automatically"
          subtitle="Started again at each reconcile, even after a manual shutdown"
          checked={autostart}
          disabled={blocked}
          onCheckedChange={(next) => draft.set({ ...draft.value, autostart: next })}
        />
      </PreferencesGroup>

      <FormActions dirty={draft.dirty} busy={busy} disabled={blocked || draft.conflict || missing} onDiscard={draft.discard} onSave={save} />
    </div>
  )
}
