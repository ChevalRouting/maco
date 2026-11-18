import { useState, type FormEvent } from 'react'
import { Button, EntryRow, PreferencesGroup, Sheet } from 'maco-ui'
import { Plus } from 'lucide-react'
import { addDisk, type VMView, type Job } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { IntegerEntryRow } from './IntegerEntryRow'
import { JobNotice } from './JobNotice'
import { VMDataDisk } from './VMDataDisk'
import { StableWidth } from './StableWidth'

interface VMDisksProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

interface AddDiskSheetProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
  onClose: () => void
}

function AddDiskSheet({ vm, run, onClose }: AddDiskSheetProps) {
  const [name, setName] = useState('')
  const [size, setSize] = useState(10)
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const valid = name.trim().length > 0 && Number.isSafeInteger(size) && size >= 1

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || !valid) return
    const job = await action.execute('vm.disk.add', () => addDisk(vm.manifest.id, name.trim(), size))
    if (job) onClose()
  }

  return (
    <Sheet open onClose={() => !action.busy && onClose()} title="Add Disk">
      <form onSubmit={add} className="space-y-5">
        <JobNotice error={action.error} />
        <PreferencesGroup>
          <EntryRow id="new-disk-name" title="Name" placeholder="data" required autoFocus value={name} onChange={(event) => setName(event.target.value)} />
          <IntegerEntryRow id="new-disk-size" title="Capacity (GiB)" min={1} required value={size} onValueChange={setSize} />
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={action.busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={action.busy || !valid}>
            <StableWidth value={action.busy ? 'Adding…' : 'Add'} values={['Add', 'Adding…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}

export function VMDisks({ vm, run }: VMDisksProps) {
  const [adding, setAdding] = useState(false)
  const disks = vm.manifest.disks || []
  const hasBoot = (vm.manifest.disk_size_gib || 0) > 0

  return (
    <div className="max-w-3xl space-y-4">
      <PreferencesGroup
        title="Disks"
        header={
          <Button variant="outline" className="gap-2" onClick={() => setAdding(true)}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Add Disk…
          </Button>
        }
      >
        {hasBoot && <VMDataDisk vm={vm} disk={{ id: 'disk', name: 'Disk 1', size_gib: vm.manifest.disk_size_gib }} primary run={run} />}
        {disks.map((disk) => (
          <VMDataDisk key={disk.id} vm={vm} disk={disk} run={run} />
        ))}
      </PreferencesGroup>
      {adding && <AddDiskSheet vm={vm} run={run} onClose={() => setAdding(false)} />}
    </div>
  )
}
