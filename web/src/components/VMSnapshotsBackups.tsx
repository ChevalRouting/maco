import { useCallback, useState, type FormEvent } from 'react'
import { Button, Dialog, EntryRow, NoticeBanner, PreferencesGroup, SwitchRow } from 'cheval-ui'
import {
  listBackups,
  createBackup,
  restoreBackup,
  deleteBackup,
  listSnapshots,
  createSnapshot,
  restoreSnapshot,
  deleteSnapshot,
  type VMView,
  type BackupInfo,
  type Snapshot,
  type Job,
} from '../api'
import { formatBytes } from '../ux'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { VMBackupSchedule } from './VMBackupSchedule'

interface VMSnapshotsBackupsProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

function backupTime(backup: BackupInfo): string {
  const value = backup.created_at ? new Date(backup.created_at) : null
  if (value && !Number.isNaN(value.getTime())) return value.toLocaleString()
  return backup.timestamp
}

export function VMSnapshotsBackups({ vm, run }: VMSnapshotsBackupsProps) {
  const id = vm.manifest.id
  const loadBackups = useCallback(() => listBackups(id), [id])
  const loadSnapshots = useCallback(() => listSnapshots(id), [id])
  const backups = useResource<BackupInfo[]>(loadBackups, [], 'vms')
  const snapshots = useResource<Snapshot[]>(loadSnapshots, [], 'vms')
  const action = useRowAction(id, vm.manifest.name, 'vm', run)
  const [restoring, setRestoring] = useState<BackupInfo | null>(null)
  const [removing, setRemoving] = useState<BackupInfo | null>(null)
  const [creating, setCreating] = useState(false)
  const [tag, setTag] = useState('')
  const [includeRAM, setIncludeRAM] = useState(true)
  const [restoringSnap, setRestoringSnap] = useState<Snapshot | null>(null)
  const [removingSnap, setRemovingSnap] = useState<Snapshot | null>(null)
  const running = vm.phase === 'running'
  const blocked = action.busy
  const tagValid = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(tag)

  async function backupNow() {
    if (blocked) return
    await action.execute('vm.backup', () => createBackup(id))
  }

  async function createSnap(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (blocked || !tagValid) return
    const job = await action.execute('vm.snapshot.create', () =>
      createSnapshot(id, tag, running && includeRAM),
    )
    if (job) { setCreating(false); setTag('') }
  }

  async function restoreSnap() {
    if (!restoringSnap || blocked) return
    const job = await action.execute('vm.snapshot.restore', () =>
      restoreSnapshot(id, restoringSnap.tag),
    )
    if (job) setRestoringSnap(null)
  }

  async function removeSnap() {
    if (!removingSnap || blocked) return
    const job = await action.execute('vm.snapshot.delete', () =>
      deleteSnapshot(id, removingSnap.tag),
    )
    if (job) setRemovingSnap(null)
  }

  async function restore(asNew: boolean) {
    if (!restoring || blocked) return
    const job = await action.execute('vm.backup.restore', () =>
      restoreBackup(id, restoring.timestamp, asNew),
    )
    if (job) setRestoring(null)
  }

  async function remove() {
    if (!removing || blocked) return
    const job = await action.execute('vm.backup.delete', () =>
      deleteBackup(id, removing.timestamp),
    )
    if (job) setRemoving(null)
  }

  return (
    <div className="space-y-8">
      <div className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-semibold">Snapshots ({snapshots.data.length})</h2>
          <Button size="sm" variant="suggested" disabled={blocked} onClick={() => setCreating(true)}>
            Create snapshot
          </Button>
        </div>
        <p className="text-sm text-muted-foreground">
          Snapshots are fast, in-place restore points stored inside the VM's disks. On a running VM
          you can capture the live RAM state so a restore resumes exactly where it left off.
        </p>
        {snapshots.error && <NoticeBanner intent="danger">{snapshots.error}</NoticeBanner>}
        {snapshots.data.length === 0 ? (
          <p className="text-sm text-muted-foreground">No snapshots yet.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-muted-foreground">
                  <th className="py-2 pr-4 font-medium">Name</th>
                  <th className="py-2 pr-4 font-medium">Created</th>
                  <th className="py-2 pr-4 font-medium">State</th>
                  <th className="py-2 pr-4 font-medium sr-only">Actions</th>
                </tr>
              </thead>
              <tbody>
                {snapshots.data.map((snapshot) => (
                  <tr key={snapshot.tag} className="border-t border-border align-middle">
                    <td className="py-2 pr-4 whitespace-nowrap font-medium">{snapshot.tag}</td>
                    <td className="py-2 pr-4 whitespace-nowrap">
                      {snapshot.created_at ? new Date(snapshot.created_at).toLocaleString() : '-'}
                    </td>
                    <td className="py-2 pr-4 whitespace-nowrap text-muted-foreground">
                      {snapshot.has_ram ? 'RAM + disk' : 'Disk only'}
                    </td>
                    <td className="py-2 pr-4">
                      <div className="flex justify-end gap-2">
                        <Button size="sm" disabled={blocked} onClick={() => setRestoringSnap(snapshot)}>
                          Restore
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          disabled={blocked}
                          onClick={() => setRemovingSnap(snapshot)}
                        >
                          Delete
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-semibold">Backups ({backups.data.length})</h2>
        <Button size="sm" variant="suggested" disabled={blocked} onClick={backupNow}>
          {action.action === 'vm.backup' ? 'Backing up…' : 'Back up now'}
        </Button>
      </div>
      <p className="text-sm text-muted-foreground">
        A backup is a full, self-contained copy of every VM disk plus its configuration. Backups can
        be created while the VM is running.
      </p>
      {backups.error && <NoticeBanner intent="danger">{backups.error}</NoticeBanner>}
      {backups.data.length === 0 ? (
        <p className="text-sm text-muted-foreground">No backups yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-muted-foreground">
                <th className="py-2 pr-4 font-medium">Created</th>
                <th className="py-2 pr-4 font-medium">Size</th>
                <th className="py-2 pr-4 font-medium">Disks</th>
                <th className="py-2 pr-4 font-medium">State</th>
                <th className="py-2 pr-4 font-medium sr-only">Actions</th>
              </tr>
            </thead>
            <tbody>
              {backups.data.map((backup) => (
                <tr key={backup.timestamp} className="border-t border-border align-middle">
                  <td className="py-2 pr-4 whitespace-nowrap">{backupTime(backup)}</td>
                  <td className="py-2 pr-4 whitespace-nowrap">{formatBytes(backup.size_bytes)}</td>
                  <td className="py-2 pr-4 whitespace-nowrap">{backup.disks.length}</td>
                  <td className="py-2 pr-4 whitespace-nowrap text-muted-foreground">
                    {backup.live ? (backup.consistent ? 'Live · consistent' : 'Live') : 'Offline'}
                  </td>
                  <td className="py-2 pr-4">
                    <div className="flex justify-end gap-2">
                      <Button size="sm" disabled={blocked} onClick={() => setRestoring(backup)}>
                        Restore
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={blocked}
                        onClick={() => setRemoving(backup)}
                      >
                        Delete
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      </div>

      <VMBackupSchedule vmID={id} />

      {restoring && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setRestoring(null) }}
          title="Restore backup"
          description={
            <>
              Restore the backup from <strong>{backupTime(restoring)}</strong>. Replacing this VM
              overwrites its current disks and requires it to be shut down first. Restoring as a new
              VM keeps this one untouched.
              {running && ' This VM is running, so stop it before replacing it in place, or restore as a new VM.'}
            </>
          }
          footer={
            <>
              <Button type="button" disabled={action.busy} onClick={() => setRestoring(null)}>
                Cancel
              </Button>
              <Button type="button" disabled={blocked || running} onClick={() => restore(false)}>
                Replace this VM
              </Button>
              <Button type="button" variant="suggested" disabled={blocked} onClick={() => restore(true)}>
                Restore as new VM
              </Button>
            </>
          }
        />
      )}

      {removing && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setRemoving(null) }}
          title="Delete backup"
          description={
            <>
              Permanently delete the backup from <strong>{backupTime(removing)}</strong>? This
              cannot be undone.
            </>
          }
          footer={
            <>
              <Button type="button" disabled={action.busy} onClick={() => setRemoving(null)}>
                Cancel
              </Button>
              <Button type="button" variant="destructive" disabled={blocked} onClick={remove}>
                {action.action === 'vm.backup.delete' ? 'Deleting…' : 'Delete backup'}
              </Button>
            </>
          }
        />
      )}

      {creating && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setCreating(false) }}
          title="Create snapshot"
        >
          <form onSubmit={createSnap} className="space-y-4">
            <fieldset disabled={blocked}>
              <PreferencesGroup title="Snapshot">
                <EntryRow
                  id="new-snapshot-name"
                  title="Name"
                  required
                  value={tag}
                  onChange={(event) => setTag(event.target.value)}
                />
                <SwitchRow
                  title="Include live RAM"
                  subtitle={
                    running
                      ? 'Capture the running memory so a restore resumes exactly where it left off.'
                      : 'The VM is not running, so only its disks are captured.'
                  }
                  checked={running && includeRAM}
                  disabled={!running}
                  onCheckedChange={setIncludeRAM}
                />
              </PreferencesGroup>
            </fieldset>
            <div className="flex justify-end gap-2">
              <Button type="button" disabled={action.busy} onClick={() => setCreating(false)}>
                Cancel
              </Button>
              <Button type="submit" variant="suggested" disabled={blocked || !tagValid}>
                {action.action === 'vm.snapshot.create' ? 'Creating…' : 'Create snapshot'}
              </Button>
            </div>
          </form>
        </Dialog>
      )}

      {restoringSnap && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setRestoringSnap(null) }}
          title="Restore snapshot"
          description={
            <>
              Restore this VM to snapshot <strong>{restoringSnap.tag}</strong>? Any changes made
              since the snapshot was taken will be lost.
              {restoringSnap.has_ram
                ? ' The live RAM state will be resumed.'
                : running
                  ? ' This snapshot has no saved RAM; restore it while the VM is stopped.'
                  : ''}
            </>
          }
          footer={
            <>
              <Button type="button" disabled={action.busy} onClick={() => setRestoringSnap(null)}>
                Cancel
              </Button>
              <Button type="button" variant="suggested" disabled={blocked} onClick={restoreSnap}>
                {action.action === 'vm.snapshot.restore' ? 'Restoring…' : 'Restore snapshot'}
              </Button>
            </>
          }
        />
      )}

      {removingSnap && (
        <Dialog
          open
          onClose={() => { if (!action.busy) setRemovingSnap(null) }}
          title="Delete snapshot"
          description={
            <>
              Permanently delete snapshot <strong>{removingSnap.tag}</strong>? This cannot be undone.
            </>
          }
          footer={
            <>
              <Button type="button" disabled={action.busy} onClick={() => setRemovingSnap(null)}>
                Cancel
              </Button>
              <Button type="button" variant="destructive" disabled={blocked} onClick={removeSnap}>
                {action.action === 'vm.snapshot.delete' ? 'Deleting…' : 'Delete snapshot'}
              </Button>
            </>
          }
        />
      )}
    </div>
  )
}
