import { IntegerEntryRow } from './IntegerEntryRow'
import { useState, type FormEvent } from 'react'
import { Button, Dialog, EntryRow, PreferencesGroup } from 'cheval-ui'
import { addDisk, type VMView, type Job } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { VMDataDisk } from './VMDataDisk'

interface VMDisksProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMDisks({ vm, run }: VMDisksProps) {
  const [adding, setAdding] = useState(false)
  const [name, setName] = useState('Disk')
  const [size, setSize] = useState(10)
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const blocked = action.busy
  const disks = vm.manifest.disks || []
  const hasBoot = (vm.manifest.disk_size_gib || 0) > 0
  const valid =
    name.trim().length > 0 && Number.isSafeInteger(size) && size >= 1

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (blocked || !valid) return

    const job = await action.execute('vm.disk.add', () =>
      addDisk(vm.manifest.id, name, size),
    )
    if (job) setAdding(false)
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h2 className="font-semibold">Disks ({disks.length + (hasBoot ? 1 : 0)})</h2>
      <Button
        size="sm"
        variant="suggested"
        disabled={blocked}
        onClick={() => setAdding(true)}
      >
        Add disk
      </Button>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-2">
        {hasBoot && (
          <VMDataDisk
            key="boot"
            vm={vm}
            disk={{
              id: 'disk',
              name: 'Disk 1',
              size_gib: vm.manifest.disk_size_gib,
            }}
            primary
            run={run}
          />
        )}
        {disks.map((disk) => (
          <VMDataDisk
            key={disk.id}
            vm={vm}
            disk={disk}
            run={run}
          />
        ))}
      </div>
      {adding && (
        <Dialog open onClose={() => { if (!action.busy) setAdding(false) }} title="Add disk">
          <form onSubmit={add} className="space-y-4">
            <fieldset disabled={blocked}>
              <PreferencesGroup title="Disk">
                <EntryRow
                  id="new-disk-name"
                  title="Name"
                  required
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
                <IntegerEntryRow
                  id="new-disk-size"
                  title="Capacity (GiB)"
                  min={1}
                  required
                  value={size}
                  onValueChange={setSize}
                />
              </PreferencesGroup>
            </fieldset>
            <div className="flex justify-end gap-2">
              <Button type="button" disabled={action.busy} onClick={() => setAdding(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="suggested"
                disabled={blocked || !valid}
              >
                {action.busy ? 'Adding…' : 'Add disk'}
              </Button>
            </div>
          </form>
        </Dialog>
      )}
    </div>
  )
}
