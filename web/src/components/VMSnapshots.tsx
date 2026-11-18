import { useCallback, useState, type FormEvent } from 'react'
import { AlertDialog, Button, EntryRow, PreferencesGroup, Row, Sheet, SwitchRow } from 'maco-ui'
import { Plus, Trash2 } from 'lucide-react'
import { listSnapshots, createSnapshot, restoreSnapshot, deleteSnapshot, type VMView, type Snapshot, type Job } from '../api'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { ActionButton } from './ActionButton'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { StableWidth } from './StableWidth'

interface VMSnapshotsProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

const tagPattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/

interface CreateSnapshotSheetProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
  onClose: () => void
}

function CreateSnapshotSheet({ vm, run, onClose }: CreateSnapshotSheetProps) {
  const [tag, setTag] = useState('')
  const [includeRAM, setIncludeRAM] = useState(true)
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const running = vm.phase === 'running'
  const valid = tagPattern.test(tag)

  async function create(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || !valid) return
    const job = await action.execute('vm.snapshot.create', () => createSnapshot(vm.manifest.id, tag, running && includeRAM))
    if (job) onClose()
  }

  return (
    <Sheet open onClose={() => !action.busy && onClose()} title="Create Snapshot">
      <form onSubmit={create} className="space-y-5">
        <JobNotice error={action.error} />
        <PreferencesGroup>
          <EntryRow
            id="new-snapshot-name"
            title="Name"
            placeholder="before-upgrade"
            required
            autoFocus
            className="font-mono"
            value={tag}
            error={tag && !valid ? 'Letters, digits, dots, underscores and hyphens' : ''}
            onChange={(event) => setTag(event.target.value)}
          />
          <SwitchRow
            title="Include Memory"
            subtitle={running ? 'Restoring resumes the running guest' : 'Only available while running'}
            checked={running && includeRAM}
            disabled={!running}
            onCheckedChange={setIncludeRAM}
          />
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={action.busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={action.busy || !valid}>
            <StableWidth value={action.busy ? 'Creating…' : 'Create'} values={['Create', 'Creating…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}

export function VMSnapshots({ vm, run }: VMSnapshotsProps) {
  const id = vm.manifest.id
  const load = useCallback(() => listSnapshots(id), [id])
  const snapshots = useResource<Snapshot[]>(load, [], 'vms')
  const action = useRowAction(id, vm.manifest.name, 'vm', run)
  const [creating, setCreating] = useState(false)
  const [restoring, setRestoring] = useState<Snapshot | null>(null)
  const [removing, setRemoving] = useState<Snapshot | null>(null)
  const running = vm.phase === 'running'

  async function restore() {
    if (!restoring) return
    const job = await action.execute('vm.snapshot.restore', () => restoreSnapshot(id, restoring.tag))
    if (job) setRestoring(null)
  }

  async function remove() {
    if (!removing) return
    const job = await action.execute('vm.snapshot.delete', () => deleteSnapshot(id, removing.tag))
    if (job) setRemoving(null)
  }

  return (
    <div className="max-w-3xl space-y-4">
      <ResourceNotice resource={snapshots} name="snapshots" />
      <JobNotice error={action.error} />
      <PreferencesGroup
        title="Snapshots"
        header={
          <Button variant="outline" className="gap-2" disabled={action.busy} onClick={() => setCreating(true)}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Create Snapshot…
          </Button>
        }
      >
        {!snapshots.loading && snapshots.data.length === 0 && <Row subtitle="None" />}
        {snapshots.data.map((snapshot) => (
          <Row
            key={snapshot.tag}
            title={<span className="font-mono">{snapshot.tag}</span>}
            subtitle={`${snapshot.created_at ? new Date(snapshot.created_at).toLocaleString() : 'Unknown date'} · ${snapshot.has_ram ? 'disks and memory' : 'disks'}`}
          >
            <Button disabled={action.busy} onClick={() => setRestoring(snapshot)}>
              Restore…
            </Button>
            <ActionButton label={`Delete ${snapshot.tag}`} icon={Trash2} disabled={action.busy} onClick={() => setRemoving(snapshot)} />
          </Row>
        ))}
      </PreferencesGroup>

      {creating && <CreateSnapshotSheet vm={vm} run={run} onClose={() => setCreating(false)} />}
      <AlertDialog
        open={!!restoring}
        onCancel={() => setRestoring(null)}
        onConfirm={restore}
        busy={action.busy}
        destructive
        title={`Restore ${restoring?.tag ?? ''}?`}
        description={
          restoring && !restoring.has_ram && running
            ? 'Shut down first; this snapshot has no memory state'
            : 'Changes made since the snapshot are lost'
        }
        confirmLabel="Restore"
      />
      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title={`Delete ${removing?.tag ?? ''}?`}
        confirmLabel="Delete Snapshot"
      />
    </div>
  )
}
