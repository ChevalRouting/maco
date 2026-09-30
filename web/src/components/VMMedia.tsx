import { useCallback, useEffect, useState } from 'react'
import {
  Badge,
  Button,
  NoticeBanner,
  PreferencesGroup,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SwitchRow,
} from 'cheval-ui'
import { ArrowDown, ArrowUp, Disc3, HardDrive, Plus, Trash2 } from 'lucide-react'
import { Link } from 'react-router-dom'
import { listMedia, updateMedia, updateHardware, errorMessage, type Media, type VMView } from '../api'
import { bootOrder } from '../bootOrder'
import { useDraft } from '../hooks/useDraft'
import { automaticStartupHelp } from '../ux'
import { JobNotice } from './JobNotice'

interface IsoSelectProps {
  value?: string
  placeholder: string
  options: Media[]
  loading: boolean
  error: string
  disabled: boolean
  ariaLabel: string
  onOpen: () => void
  onChange: (value: string) => void
  onRetry: () => void
}

function IsoSelect({
  value,
  placeholder,
  options,
  loading,
  error,
  disabled,
  ariaLabel,
  onOpen,
  onChange,
  onRetry,
}: IsoSelectProps) {
  return (
    <Select
      value={value}
      disabled={disabled}
      onValueChange={onChange}
      onOpenChange={(open) => {
        if (open) onOpen()
      }}
    >
      <SelectTrigger aria-label={ariaLabel} className="mt-1 w-full">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent className="z-[60] max-w-[calc(100vw-2rem)]">
        {loading && (
          <div className="px-3 py-2 text-sm text-muted-foreground">Loading ISOs…</div>
        )}
        {error && (
          <div className="px-3 py-2 text-sm text-destructive">
            Couldn’t load ISOs.{' '}
            <button type="button" className="underline" onClick={onRetry}>
              Retry
            </button>
          </div>
        )}
        {!loading && !error && options.map((iso) => (
          <SelectItem key={iso.id} value={iso.id} className="whitespace-normal break-words">
            {iso.name}
          </SelectItem>
        ))}
        {!loading && !error && !options.length && (
          <div className="px-3 py-2 text-sm text-muted-foreground">
            No ISOs available.{' '}
            <Link className="text-primary underline" to="/media">
              Upload an ISO
            </Link>
          </div>
        )}
      </SelectContent>
    </Select>
  )
}

export function VMMedia({ vm, onSaved }: { vm: VMView; onSaved: () => void }) {
  const draft = useDraft(`maco:draft:${vm.manifest.id}:media`, {
    isos: vm.manifest.isos || [],
    order: bootOrder(vm.manifest),
    autostart: !!vm.manifest.autostart,
  })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [adding, setAdding] = useState(false)
  const [isoList, setIsoList] = useState<Media[] | null>(null)
  const [loadingIsos, setLoadingIsos] = useState(false)
  const [isoError, setIsoError] = useState('')

  const { isos, autostart } = draft.value
  const order = bootOrder({ ...vm.manifest, boot_order: draft.value.order }, isos)
  const blocked = vm.phase === 'running' || busy

  const ensureIsos = useCallback(async () => {
    if (isoList !== null || loadingIsos) return
    setLoadingIsos(true)
    setIsoError('')
    try {
      const list = await listMedia()
      setIsoList(list.filter((item) => item.kind === 'iso'))
    } catch (e) {
      setIsoError(errorMessage(e))
    } finally {
      setLoadingIsos(false)
    }
  }, [isoList, loadingIsos])

  useEffect(() => {
    if ((vm.manifest.isos || []).length) ensureIsos()
  }, [ensureIsos, vm.manifest.isos])

  function isoName(id: string): string {
    const found = isoList?.find((item) => item.id === id)
    if (found) return found.name
    return isoList ? 'Unavailable ISO' : 'Loading…'
  }

  function choicesFor(selfId?: string): Media[] {
    return (isoList || []).filter((item) => item.id === selfId || !isos.includes(item.id))
  }

  const devices = [
    { id: 'disk', name: 'Disk 1', detail: `${vm.manifest.disk_size_gib} GiB`, iso: false },
    ...(vm.manifest.disks || []).map((disk) => ({
      id: `disk:${disk.id}`,
      name: disk.name,
      detail: `${disk.size_gib} GiB`,
      iso: false,
    })),
    ...isos.map((id) => ({
      id: `iso:${id}`,
      name: isoName(id),
      detail: 'CD-ROM',
      iso: true,
    })),
  ]

  function addIso(id: string) {
    draft.set({ ...draft.value, isos: [...isos, id], order: [...order, `iso:${id}`] })
    setAdding(false)
    setSaved(false)
  }

  function removeIso(id: string) {
    draft.set({
      ...draft.value,
      isos: isos.filter((item) => item !== id),
      order: order.filter((item) => item !== `iso:${id}`),
    })
    setSaved(false)
  }

  function changeIso(oldId: string, newId: string) {
    if (oldId === newId) return
    draft.set({
      ...draft.value,
      isos: isos.map((item) => (item === oldId ? newId : item)),
      order: order.map((item) => (item === `iso:${oldId}` ? `iso:${newId}` : item)),
    })
    setSaved(false)
  }

  function move(index: number, delta: number) {
    const next = [...order]
    ;[next[index], next[index + delta]] = [next[index + delta], next[index]]
    draft.set({ ...draft.value, order: next })
    setSaved(false)
  }

  function setAutostart(next: boolean) {
    draft.set({ ...draft.value, autostart: next })
    setSaved(false)
  }

  const unavailable = isoList ? isos.filter((id) => !isoList.some((item) => item.id === id)) : []

  async function save() {
    if (blocked || !draft.dirty || draft.conflict) return
    setBusy(true)
    setError('')
    setSaved(false)
    try {
      const baseOrder = bootOrder(vm.manifest)
      const mediaChanged =
        JSON.stringify(isos) !== JSON.stringify(vm.manifest.isos || []) ||
        JSON.stringify(order) !== JSON.stringify(baseOrder)
      if (mediaChanged) await updateMedia(vm.manifest.id, isos, order)
      if (autostart !== !!vm.manifest.autostart) {
        await updateHardware(vm.manifest.id, { autostart })
      }
      onSaved()
      setSaved(true)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-3xl space-y-6">
      <JobNotice error={error} />

      {vm.phase === 'running' && (
        <NoticeBanner intent="info">
          Shut down the VM before changing CD-ROMs, boot order, or automatic startup. Your draft is kept.
        </NoticeBanner>
      )}
      {draft.conflict && (
        <NoticeBanner intent="warning">
          Saved media or disks changed while you were editing. Discard this draft to load the latest configuration.
        </NoticeBanner>
      )}

      <section className="space-y-4" aria-label="Saved Boot Configuration">
        <p className="text-sm text-muted-foreground">
          CD-ROM drives, boot order, and automatic startup are saved together. Boot first chooses the first device the guest attempts.
        </p>

        <PreferencesGroup title="CD-ROM Drives">
          {isos.map((id, index) => (
            <div key={`${id}:${index}`} className="flex flex-wrap items-center gap-3 p-4">
              <Disc3 aria-hidden="true" className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-[10rem] flex-1">
                <p className="text-sm font-medium">CD-ROM {index + 1}</p>
                <IsoSelect
                  value={isoList?.some((item) => item.id === id) ? id : undefined}
                  placeholder={isoName(id)}
                  options={choicesFor(id)}
                  loading={loadingIsos}
                  error={isoError}
                  disabled={blocked}
                  ariaLabel={`CD-ROM ${index + 1} ISO`}
                  onOpen={ensureIsos}
                  onChange={(next) => changeIso(id, next)}
                  onRetry={ensureIsos}
                />
              </div>
              <Button
                size="sm"
                variant="outline"
                disabled={blocked}
                onClick={() => removeIso(id)}
                aria-label={`Remove CD-ROM ${index + 1}`}
              >
                <Trash2 aria-hidden="true" className="h-4 w-4" />
              </Button>
            </div>
          ))}

          {adding && (
            <div className="flex flex-wrap items-center gap-3 p-4">
              <Disc3 aria-hidden="true" className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-[10rem] flex-1">
                <p className="text-sm font-medium">CD-ROM {isos.length + 1}</p>
                <IsoSelect
                  placeholder="Choose an ISO"
                  options={choicesFor()}
                  loading={loadingIsos}
                  error={isoError}
                  disabled={blocked}
                  ariaLabel="Choose an ISO for the new CD-ROM"
                  onOpen={ensureIsos}
                  onChange={addIso}
                  onRetry={ensureIsos}
                />
              </div>
              <Button
                size="sm"
                variant="outline"
                disabled={busy}
                onClick={() => setAdding(false)}
                aria-label="Cancel new CD-ROM"
              >
                Cancel
              </Button>
            </div>
          )}

          {!isos.length && !adding && (
            <p className="p-4 text-sm text-muted-foreground">
              No CD-ROM drives. Add a CD-ROM to attach an installation ISO.
            </p>
          )}

          <div className="p-4">
            <Button
              size="sm"
              variant="outline"
              disabled={blocked || adding}
              onClick={() => {
                setAdding(true)
                ensureIsos()
              }}
            >
              <Plus aria-hidden="true" className="mr-2 h-4 w-4" />
              Add CD-ROM
            </Button>
          </div>
        </PreferencesGroup>

        <PreferencesGroup title="Boot Order">
          {order.map((id, index) => {
            const device = devices.find((item) => item.id === id)!
            const Icon = device.iso ? Disc3 : HardDrive
            return (
              <div key={id} className="flex flex-wrap items-center gap-2 p-4">
                <span className="w-5 text-sm tabular-nums text-muted-foreground">{index + 1}</span>
                <Icon aria-hidden="true" className="h-5 w-5 text-muted-foreground" />
                <div className="min-w-[8rem] flex-1">
                  <p className="break-words font-medium">{device.name}</p>
                  <p className="text-sm text-muted-foreground">{device.detail}</p>
                </div>
                {index === 0 && <Badge variant="neutral">Boot First</Badge>}
                <Button
                  size="sm"
                  variant="outline"
                  disabled={blocked || index === 0}
                  onClick={() => move(index, -1)}
                  aria-label={`Move ${device.name} up`}
                >
                  <ArrowUp aria-hidden="true" className="h-4 w-4" />
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={blocked || index === order.length - 1}
                  onClick={() => move(index, 1)}
                  aria-label={`Move ${device.name} down`}
                >
                  <ArrowDown aria-hidden="true" className="h-4 w-4" />
                </Button>
              </div>
            )
          })}
        </PreferencesGroup>

        <PreferencesGroup title="Startup">
          <SwitchRow
            title="Automatic Startup"
            subtitle={automaticStartupHelp}
            checked={autostart}
            disabled={blocked}
            onCheckedChange={setAutostart}
          />
        </PreferencesGroup>

        <div className="flex flex-wrap items-center gap-3">
          <Button
            variant="suggested"
            disabled={blocked || !draft.dirty || draft.conflict || unavailable.length > 0}
            onClick={save}
          >
            {busy ? 'Saving…' : 'Save Boot Configuration'}
          </Button>
          <Button
            variant="outline"
            disabled={busy || !draft.dirty}
            onClick={() => {
              draft.discard()
              setAdding(false)
              setSaved(false)
            }}
          >
            Discard Changes
          </Button>
          <p role="status" className="text-sm text-muted-foreground">
            {saved
              ? 'Boot configuration saved'
              : draft.dirty
                ? 'Unsaved changes · Draft kept in this browser tab'
                : 'Matches saved configuration'}
          </p>
        </div>

        {unavailable.length > 0 && (
          <NoticeBanner intent="warning">
            A selected ISO is no longer in the library. Choose an available ISO or remove that CD-ROM before saving.
          </NoticeBanner>
        )}
      </section>
    </div>
  )
}
