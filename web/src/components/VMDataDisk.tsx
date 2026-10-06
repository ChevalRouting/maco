import { IntegerEntryRow } from './IntegerEntryRow'
import { useState, type FormEvent } from 'react'
import { AlertDialog, Badge, Button, Dialog, PreferencesGroup } from 'cheval-ui'
import { HardDrive } from 'lucide-react'
import {
  growDisk,
  removeDisk,
  updateHardware,
  type VMDisk,
  type VMView,
  type Job,
} from '../api'
import { bootOrder } from '../bootOrder'
import { useRowAction } from '../hooks/useRowAction'

interface VMDataDiskProps {
  vm: VMView
  disk: VMDisk
  primary?: boolean
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMDataDisk({ vm, disk, primary, run }: VMDataDiskProps) {
  const [size, setSize] = useState(disk.size_gib + 1)
  const [growing, setGrowing] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const canGrow = Number.isSafeInteger(size) && size > disk.size_gib

  async function grow(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || !canGrow) return

    const job = await action.execute(
      primary ? 'vm.hardware' : 'vm.disk.grow',
      () =>
        primary
          ? updateHardware(vm.manifest.id, { disk_size_gib: size })
          : growDisk(vm.manifest.id, disk.id, size),
    )
    if (job) setGrowing(false)
  }

  async function remove() {
    if (action.busy || vm.phase === 'running') return

    const job = await action.execute('vm.disk.remove', () =>
      removeDisk(vm.manifest.id, disk.id),
    )
    if (job) setConfirm(false)
  }

  return (
    <>
      <article
        className="flex min-w-0 flex-col rounded-xl border bg-card p-3"
        aria-label={disk.name}
      >
        <div className="flex items-center justify-between gap-2">
          <HardDrive
            aria-hidden="true"
            className="h-5 w-5 text-muted-foreground"
          />
          {bootOrder(vm.manifest)[0] === (primary ? 'disk' : `disk:${disk.id}`) && (
            <Badge variant="neutral">Boot first</Badge>
          )}
        </div>
        <h3 className="mt-2 break-words font-semibold" title={disk.name}>
          {disk.name}
        </h3>
        <p
          className="mt-1 truncate text-xl font-semibold"
          title={`${disk.size_gib} GiB`}
        >
          {disk.size_gib}{' '}
          <span className="text-sm font-normal text-muted-foreground">GiB</span>
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          QCOW2 · {primary ? 'VirtIO' : 'SCSI'}
        </p>
        <div className="mt-auto pt-2">
          <div className="flex gap-1">
            <Button
              size="sm"
              disabled={action.busy}
              onClick={() => {
                setSize(disk.size_gib + 1)
                setGrowing(true)
              }}
              aria-haspopup="dialog"
            >
              Increase Capacity…
            </Button>
            {!primary && (
              <Button
                size="sm"
                variant="outline"
                disabled={action.busy || vm.phase === 'running'}
                onClick={() => setConfirm(true)}
                title={
                  vm.phase === 'running'
                    ? 'Stop the VM before removing this disk'
                    : 'Remove disk'
                }
              >
                Delete…
              </Button>
            )}
          </div>
        </div>
      </article>
      {growing && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setGrowing(false) }}
          title={`Increase ${disk.name} Capacity`}
          description="Capacity can only increase. Afterward, expand the partition and filesystem inside the guest to use the additional space."
        >
          <form onSubmit={grow} className="space-y-4">
            <PreferencesGroup title={`Current capacity: ${disk.size_gib} GiB`}>
              <IntegerEntryRow
                id={`disk-${disk.id}`}
                title="New capacity (GiB)"
                min={disk.size_gib + 1}
                required
                value={size}
                onValueChange={setSize}
                disabled={action.busy}
              />
            </PreferencesGroup>
            <div className="flex justify-end gap-2">
              <Button type="button" disabled={action.busy} onClick={() => setGrowing(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="suggested"
                disabled={action.busy || !canGrow}
              >
                {action.busy ? 'Increasing…' : 'Increase Capacity'}
              </Button>
            </div>
          </form>
        </Dialog>
      )}
      <AlertDialog
        open={confirm}
        onCancel={() => setConfirm(false)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title={`Delete ${disk.name}?`}
        description="This permanently deletes this disk and all data on it."
        confirmLabel="Delete Disk"
      />
    </>
  )
}
