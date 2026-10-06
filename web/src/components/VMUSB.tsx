import { useCallback, useState } from 'react'
import { AlertDialog, Badge, Button, Dialog, EntryRow, PreferencesGroup } from 'cheval-ui'
import { Usb } from 'lucide-react'
import { assignUSB, unassignUSB, attachUSB, detachUSB, listUSBDevices, listVMUSB, type Job, type USBDevice, type USBInventory, type USBAttachment, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'

interface VMUSBProps {
  error?: string
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

function hex(value: number) { return value.toString(16).padStart(4, '0') }

function identity(device: USBDevice) {
  const parts = []
  if (device.serial) parts.push(`Serial ${device.serial}`)
  if (device.port) parts.push(`Port ${device.bus}/${device.port}`)
  if (!parts.length) parts.push(`${hex(device.vendor_id)}:${hex(device.product_id)}`)
  return parts.join(' · ')
}

function USBIdentity({ device }: { device: USBDevice }) {
  return (
    <span className="flex min-w-0 items-center gap-3">
      <Usb aria-hidden="true" className="h-5 w-5 shrink-0 text-muted-foreground" />
      <span className="min-w-0 whitespace-normal break-words">
        <span className="block font-medium">{device.product}</span>
        <span className="block text-sm font-normal text-muted-foreground">{identity(device)}</span>
      </span>
    </span>
  )
}

function DeviceDetails({ device }: { device: USBDevice }) {
  return (
    <details className="text-sm text-muted-foreground">
      <summary className="cursor-pointer">Device details</summary>
      <dl className="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 break-words">
        <dt>Manufacturer</dt><dd>{device.manufacturer || 'Unavailable'}</dd>
        <dt>USB ID</dt><dd>{hex(device.vendor_id)}:{hex(device.product_id)}</dd>
        <dt>Serial</dt><dd>{device.serial || 'Unavailable'}</dd>
        {device.port && <><dt>Port</dt><dd>{device.bus}/{device.port}</dd></>}
        {!!device.classes?.length && <><dt>Type</dt><dd>{device.classes.join(', ')}</dd></>}
      </dl>
    </details>
  )
}

function attachmentLabel(state: string) {
  if (state === 'attached') return 'Connected'
  if (state === 'on-start') return 'On startup'
  if (state === 'missing') return 'Disconnected'
  return 'Check connection'
}

export function VMUSB({ vm, run, error = '' }: VMUSBProps) {
  const [adding, setAdding] = useState(false)
  const [removing, setRemoving] = useState<USBAttachment | null>(null)
  const load = useCallback(() => listVMUSB(vm.manifest.id), [vm.manifest.id])
  const attachments = useResource<USBAttachment[]>(load, [], 'usb')
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const running = vm.phase === 'running'

  async function attach(device: USBDevice) {
    if (action.busy) return
    const job = await action.execute(running ? 'vm.usb.attach' : 'vm.usb.assign', () =>
      running ? attachUSB(vm.manifest.id, device) : assignUSB(vm.manifest.id, device),
    )
    if (job) setAdding(false)
  }

  async function remove(attachment: USBAttachment) {
    if (action.busy) return
    const job = await action.execute(attachment.assignment_key ? 'vm.usb.unassign' : 'vm.usb.detach', () =>
      attachment.assignment_key
        ? unassignUSB(vm.manifest.id, attachment.assignment_key)
        : detachUSB(vm.manifest.id, attachment.id),
    )
    if (job) setRemoving(null)
  }

  function requestRemove(attachment: USBAttachment) {
    if (attachment.state === 'attached' && attachment.device.classes?.includes('Storage')) setRemoving(attachment)
    else void remove(attachment)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {running
            ? 'Connect devices for this running session. Temporary connections end when the VM stops.'
            : 'Assign devices to connect automatically on every startup while they remain plugged in.'}
        </p>
        <Button variant="suggested" disabled={action.busy} onClick={() => setAdding(true)}>
          {running ? 'Connect Device…' : 'Assign on Startup…'}
        </Button>
      </div>

      <ResourceNotice resource={attachments} name="USB connections" />

      {!attachments.loading && !attachments.error && attachments.data.length === 0 && (
        <p className="py-6 text-center text-sm text-muted-foreground">No USB devices added.</p>
      )}

      <div className="space-y-2">
        {attachments.data.map((attachment) => (
          <article key={attachment.id} className="space-y-3 rounded-xl border bg-card p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <USBIdentity device={attachment.device} />
              <div className="flex items-center gap-3">
                <Badge
                  variant={
                    attachment.state === 'attached'
                      ? 'success'
                      : attachment.state === 'on-start'
                        ? 'neutral'
                        : 'warning'
                  }
                >
                  {attachmentLabel(attachment.state)}
                </Badge>
                <Button size="sm" disabled={action.busy} onClick={() => requestRemove(attachment)}>
                  {attachment.assignment_key ? 'Remove Startup Assignment' : 'Disconnect'}
                </Button>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">
              {attachment.assignment_key
                ? 'Assigned for every startup. Removing the assignment also disconnects an active device.'
                : 'Temporary connection for this VM session.'}
            </p>
            <DeviceDetails device={attachment.device} />
            {attachment.reason && <p className="text-sm">{attachment.reason}</p>}
          </article>
        ))}
      </div>

      {adding && (
        <USBPicker
          error={error}
          busy={action.busy}
          running={running}
          added={attachments.data}
          onClose={() => setAdding(false)}
          onAttach={attach}
        />
      )}

      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={() => removing && void remove(removing)}
        busy={action.busy}
        title={`Detach ${removing?.device.product || 'drive'}?`}
        description="Eject the drive in the VM first."
        confirmLabel="Detach"
      />
    </div>
  )
}

interface USBPickerProps {
  error: string
  busy: boolean
  running: boolean
  added: USBAttachment[]
  onClose: () => void
  onAttach: (device: USBDevice) => Promise<void>
}

function USBPicker({ busy, running, added, onClose, onAttach, error }: USBPickerProps) {
  const [search, setSearch] = useState('')
  const [refresh, setRefresh] = useState(0)
  const [selected, setSelected] = useState<USBDevice | null>(null)
  const load = useCallback(() => listUSBDevices(), [refresh])
  const inventory = useResource<USBInventory>(load, { devices: [], supported: false }, 'usb')
  function alreadyAdded(device: USBDevice) {
    return added.some(
      (entry) =>
        entry.device.id === device.id ||
        (entry.assignment_key &&
          entry.device.vendor_id === device.vendor_id &&
          entry.device.product_id === device.product_id &&
          (!entry.device.serial || entry.device.serial === device.serial)),
    )
  }

  const current = inventory.data.devices.find(
    (device) =>
      device.id === selected?.id &&
      device.fingerprint === selected.fingerprint &&
      device.state === 'available' &&
      !alreadyAdded(device),
  )
  const query = search.trim().toLowerCase()
  const devices = inventory.data.devices.filter((device) =>
    `${device.product} ${device.manufacturer} ${device.serial} ${hex(device.vendor_id)}:${hex(device.product_id)} ${device.bus}/${device.port} ${device.classes.join(' ')}`
      .toLowerCase()
      .includes(query),
  )
  const blocked = busy || !!inventory.error || !inventory.data.supported

  return (
    <Dialog
      open
      onClose={() => {
        if (!busy) onClose()
      }}
      title={running ? 'Connect USB Device' : 'Assign USB Device on Startup'}
      className="mx-0 max-w-lg"
      footer={
        <>
          <Button disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="suggested"
            disabled={blocked || !current}
            onClick={() => current && void onAttach(current)}
          >
            {busy ? 'Connecting…' : running ? 'Connect for This Session' : 'Assign on Startup'}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <PreferencesGroup>
              <EntryRow
                id="usb-search"
                title="Search"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Name or serial"
              />
            </PreferencesGroup>
          </div>
          <Button disabled={busy} onClick={() => setRefresh((value) => value + 1)}>
            Refresh
          </Button>
        </div>

        <ResourceNotice resource={inventory} name="USB devices" />
        <JobNotice error={error} />

        {!inventory.loading && !inventory.error && !inventory.data.supported && (
          <p className="text-sm text-muted-foreground">
            {inventory.data.reason || 'USB passthrough is unavailable on this host.'}
          </p>
        )}
        {!inventory.loading && !inventory.error && inventory.data.supported && devices.length === 0 && (
          <p className="py-4 text-center text-sm text-muted-foreground">
            {query ? 'No matching devices.' : 'Plug a USB device into this Mac.'}
          </p>
        )}

        <fieldset disabled={blocked} className="max-h-80 space-y-2 overflow-y-auto">
          <legend className="mb-2 text-sm font-medium">Choose One Device</legend>
          {devices.map((device) => (
            <div key={device.id}>
              <label className="flex min-h-12 items-center gap-3 rounded-lg border p-3 focus-within:ring-2 focus-within:ring-inset focus-within:ring-ring">
                <input
                  type="radio"
                  name="usb-device"
                  checked={current?.id === device.id}
                  aria-label={`${device.product}, ${identity(device)}`}
                  disabled={blocked || device.state !== 'available' || alreadyAdded(device)}
                  onChange={() => setSelected(device)}
                  className="h-4 w-4 shrink-0 accent-[var(--accent)]"
                />
                <USBIdentity device={device} />
              </label>
              {(device.reason || alreadyAdded(device)) && (
                <p className="px-3 pt-1 text-sm text-muted-foreground">
                  {alreadyAdded(device) ? 'Already added' : device.reason}
                </p>
              )}
            </div>
          ))}
        </fieldset>

        {selected && !current && (
          <p role="status" className="text-sm">
            Device changed. Select it again.
          </p>
        )}
        {current && <DeviceDetails device={current} />}

        <p className="text-sm text-muted-foreground">
          {running
            ? 'Connects now for this session only. The host cannot use the device while this VM owns it.'
            : 'Connects on every startup. Keep it plugged in; the host cannot use it while this VM owns it.'}
          {current?.classes.includes('Storage') && running ? ' Eject it from this Mac first.' : ''}
        </p>
      </div>
    </Dialog>
  )
}
