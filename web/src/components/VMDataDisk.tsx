import { useState, type FormEvent } from 'react'
import { AlertDialog, Badge, Button, Dialog, PreferencesGroup, Row, type MenuItem } from 'maco-ui'
import { Eraser, HardDrive, ImageUp, Maximize2, MoreVertical, Trash2 } from 'lucide-react'
import { growDisk, removeDisk, wipeDisk, replaceBootDisk, updateHardware, type VMDisk, type VMView, type Job } from '../api'
import { bootOrder } from '../bootOrder'
import { useRowAction } from '../hooks/useRowAction'
import { IntegerEntryRow } from './IntegerEntryRow'
import { JobNotice } from './JobNotice'
import { ReplaceBootDiskDialog } from './ReplaceBootDiskDialog'
import { Menu } from './Menu'
import { StableWidth } from './StableWidth'

type DiskDialog = 'grow' | 'wipe' | 'delete' | 'replace'

interface VMDataDiskProps {
  vm: VMView
  disk: VMDisk
  primary?: boolean
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMDataDisk({ vm, disk, primary, run }: VMDataDiskProps) {
  const [open, setOpen] = useState<DiskDialog | null>(null)
  const [size, setSize] = useState(disk.size_gib + 1)
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const running = vm.phase === 'running'
  const bootsFirst = bootOrder(vm.manifest)[0] === (primary ? 'disk' : `disk:${disk.id}`)

  function close() {
    if (!action.busy) setOpen(null)
  }

  function startGrow() {
    setSize(disk.size_gib + 1)
    setOpen('grow')
  }

  async function grow(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || size <= disk.size_gib) return
    const job = await action.execute(primary ? 'vm.hardware' : 'vm.disk.grow', () =>
      primary ? updateHardware(vm.manifest.id, { disk_size_gib: size }) : growDisk(vm.manifest.id, disk.id, size),
    )
    if (job) setOpen(null)
  }

  async function confirm() {
    const job =
      open === 'wipe'
        ? await action.execute('vm.disk.wipe', () => wipeDisk(vm.manifest.id, disk.id))
        : await action.execute('vm.disk.remove', () => removeDisk(vm.manifest.id, disk.id))
    if (job) setOpen(null)
  }

  async function replace(imageId: string) {
    const job = await action.execute('vm.disk.replace', () => replaceBootDisk(vm.manifest.id, imageId))
    if (job) setOpen(null)
  }

  const sections: MenuItem[][] = [
    [
      { key: 'grow', label: 'Increase Capacity…', icon: Maximize2, onSelect: startGrow },
      ...(primary ? [{ key: 'replace', label: 'Replace from Image…', icon: ImageUp, disabled: running, onSelect: () => setOpen('replace') }] : []),
    ],
    [
      { key: 'wipe', label: 'Wipe…', icon: Eraser, destructive: true, disabled: running, onSelect: () => setOpen('wipe') },
      { key: 'delete', label: 'Delete…', icon: Trash2, destructive: true, disabled: running, onSelect: () => setOpen('delete') },
    ],
  ]

  return (
    <>
      <Row
        prefix={<HardDrive className="h-5 w-5" />}
        title={
          <span className="flex flex-wrap items-center gap-2">
            {disk.name}
            {bootsFirst && <Badge variant="neutral">Boot</Badge>}
          </span>
        }
        subtitle={`${disk.size_gib} GiB · QCOW2 · ${primary ? 'VirtIO' : 'SCSI'}`}
      >
        <Menu label={`Actions for ${disk.name}`} sections={sections} className="h-8 w-8">
          <MoreVertical aria-hidden="true" className="h-4 w-4" />
        </Menu>
      </Row>

      {open === 'grow' && (
        <Dialog
          open
          onClose={close}
          title={`Increase ${disk.name} Capacity`}
          description="Grow the guest partition and filesystem afterward"
        >
          <form onSubmit={grow} className="space-y-4">
            <JobNotice error={action.error} />
            <PreferencesGroup>
              <IntegerEntryRow
                id={`disk-${disk.id}`}
                title="Capacity (GiB)"
                min={disk.size_gib + 1}
                required
                value={size}
                help={`Currently ${disk.size_gib} GiB`}
                onValueChange={setSize}
                disabled={action.busy}
              />
            </PreferencesGroup>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" disabled={action.busy} onClick={close}>
                Cancel
              </Button>
              <Button type="submit" variant="suggested" disabled={action.busy || size <= disk.size_gib}>
                <StableWidth value={action.busy ? 'Increasing…' : 'Increase'} values={['Increase', 'Increasing…']} />
              </Button>
            </div>
          </form>
        </Dialog>
      )}
      {open === 'replace' && (
        <ReplaceBootDiskDialog vm={vm} busy={action.busy} error={action.error} onClose={close} onReplace={replace} />
      )}
      <AlertDialog
        open={open === 'wipe'}
        onCancel={close}
        onConfirm={confirm}
        busy={action.busy}
        destructive
        title={`Wipe ${disk.name}?`}
        description={primary ? 'All data, including the operating system, is erased; the capacity is kept' : 'All data is erased; the capacity is kept'}
        confirmLabel="Wipe Disk"
      />
      <AlertDialog
        open={open === 'delete'}
        onCancel={close}
        onConfirm={confirm}
        busy={action.busy}
        destructive
        title={`Delete ${disk.name}?`}
        description={primary ? 'The VM is left without a boot disk' : 'All data on the disk is permanently deleted'}
        confirmLabel="Delete Disk"
      />
    </>
  )
}
