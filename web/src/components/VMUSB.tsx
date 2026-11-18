import { useCallback, useState } from 'react'
import { AlertDialog, Button, Dialog, EmptyState, SearchInput, PreferencesGroup, ReloadButton, Row, StatusChip, type StatusIntent } from 'maco-ui'
import { Plus, Trash2, Usb } from 'lucide-react'
import { assignUSB, unassignUSB, attachUSB, detachUSB, listUSBDevices, listVMUSB, type Job, type USBDevice, type USBInventory, type USBAttachment, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { ActionButton } from './ActionButton'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'

interface VMUSBProps {
  error?: string
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

function hex(value: number) {
  return value.toString(16).padStart(4, '0')
}

function identity(device: USBDevice) {
  const parts = [`${hex(device.vendor_id)}:${hex(device.product_id)}`]
  if (device.serial) parts.push(device.serial)
  if (device.port) parts.push(`port ${device.bus}/${device.port}`)
  return parts.join(' · ')
}

interface AttachmentState {
  label: string
  intent: StatusIntent
}

function attachmentState(state: string): AttachmentState {
  if (state === 'attached') return { label: 'Connected', intent: 'success' }
  if (state === 'on-start') return { label: 'At Start', intent: 'neutral' }
  if (state === 'missing') return { label: 'Unplugged', intent: 'warning' }
  return { label: 'Unknown', intent: 'warning' }
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
      attachment.assignment_key ? unassignUSB(vm.manifest.id, attachment.assignment_key) : detachUSB(vm.manifest.id, attachment.id),
    )
    if (job) setRemoving(null)
  }

  function requestRemove(attachment: USBAttachment) {
    if (attachment.state === 'attached' && attachment.device.classes?.includes('Storage')) setRemoving(attachment)
    else void remove(attachment)
  }

  return (
    <div className="max-w-3xl space-y-4">
      <ResourceNotice resource={attachments} name="USB devices" />
      <JobNotice error={action.error} />
      <PreferencesGroup
        title="USB Devices"
        description={running ? 'Devices added now stay connected until the VM stops' : 'Devices are connected at every start while plugged in'}
        header={
          <Button variant="outline" className="gap-2" disabled={action.busy} onClick={() => setAdding(true)}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Add Device…
          </Button>
        }
      >
        {!attachments.loading && !attachments.error && attachments.data.length === 0 && <Row subtitle="None" />}
        {attachments.data.map((attachment) => {
          const state = attachmentState(attachment.state)
          return (
            <Row
              key={attachment.id}
              prefix={<Usb className="h-5 w-5" />}
              title={attachment.device.product}
              subtitle={
                <>
                  <span className="block font-mono">{identity(attachment.device)}</span>
                  {attachment.reason && <span className="block">{attachment.reason}</span>}
                </>
              }
            >
              <StatusChip intent={state.intent} label={state.label} />
              <ActionButton
                label={attachment.assignment_key ? `Remove ${attachment.device.product}` : `Disconnect ${attachment.device.product}`}
                icon={Trash2}
                disabled={action.busy}
                onClick={() => requestRemove(attachment)}
              />
            </Row>
          )
        })}
      </PreferencesGroup>

      {adding && (
        <USBPicker error={error} busy={action.busy} running={running} added={attachments.data} onClose={() => setAdding(false)} onAttach={attach} />
      )}

      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={() => removing && void remove(removing)}
        busy={action.busy}
        title={`Disconnect ${removing?.device.product || 'Drive'}?`}
        description="Eject the drive in the guest first"
        confirmLabel="Disconnect"
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

  function unavailable(device: USBDevice): string {
    if (alreadyAdded(device)) return 'Already added'
    if (device.state !== 'available') return device.reason || 'Unavailable'
    return ''
  }

  const query = search.trim().toLowerCase()
  const devices = inventory.data.devices.filter((device) =>
    `${device.product} ${device.manufacturer} ${device.serial} ${hex(device.vendor_id)}:${hex(device.product_id)} ${device.classes.join(' ')}`
      .toLowerCase()
      .includes(query),
  )
  const blocked = busy || !!inventory.error || !inventory.data.supported

  return (
    <Dialog
      open
      onClose={() => !busy && onClose()}
      title={running ? 'Connect USB Device' : 'Add USB Device'}
      description={running ? 'The host loses access while the VM uses it' : 'Connected at every start; the host loses access while the VM runs'}
      footer={
        <Button variant="outline" disabled={busy} onClick={onClose}>
          Cancel
        </Button>
      }
    >
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <SearchInput aria-label="Search Devices" placeholder="Search devices…" value={search} onChange={(event) => setSearch(event.target.value)} onClear={() => setSearch('')} />
          <ReloadButton onClick={() => setRefresh((value) => value + 1)} loading={inventory.loading} />
        </div>
        <ResourceNotice resource={inventory} name="USB devices" />
        <JobNotice error={error} />
        {!inventory.loading && !inventory.error && !inventory.data.supported && (
          <EmptyState icon={<Usb />} title="USB Unavailable" message={inventory.data.reason || undefined} />
        )}
        {!inventory.loading && !inventory.error && inventory.data.supported && devices.length === 0 && (
          <EmptyState icon={<Usb />} title={query ? 'No Results' : 'No USB Devices'} />
        )}
        {devices.length > 0 && (
          <PreferencesGroup>
            {devices.map((device) => {
              const reason = unavailable(device)
              const eject = running && device.classes.includes('Storage') ? ' · eject on the host first' : ''
              return (
                <Row key={device.id} prefix={<Usb className="h-5 w-5" />} title={device.product} subtitle={reason || <span className="font-mono">{identity(device) + eject}</span>}>
                  <Button disabled={blocked || !!reason} onClick={() => void onAttach(device)}>
                    {running ? 'Connect' : 'Add'}
                  </Button>
                </Row>
              )
            })}
          </PreferencesGroup>
        )}
      </div>
    </Dialog>
  )
}
