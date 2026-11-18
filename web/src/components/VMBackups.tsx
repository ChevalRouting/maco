import { useCallback, useState } from 'react'
import { AlertDialog, Button, Dialog, PreferencesGroup, Row } from 'maco-ui'
import { Trash2 } from 'lucide-react'
import { listBackups, createBackup, restoreBackup, deleteBackup, pruneBackups, type VMView, type BackupInfo, type Job } from '../api'
import { formatBytes, plural } from '../ux'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { ActionButton } from './ActionButton'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { VMBackupSchedule } from './VMBackupSchedule'
import { StableWidth } from './StableWidth'

interface VMBackupsProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

function backupTime(backup: BackupInfo): string {
  const value = backup.created_at ? new Date(backup.created_at) : null
  if (value && !Number.isNaN(value.getTime())) return value.toLocaleString()
  return backup.timestamp
}

function backupKind(backup: BackupInfo): string {
  if (!backup.live) return 'offline'
  return backup.consistent ? 'live, consistent' : 'live'
}

export function VMBackups({ vm, run }: VMBackupsProps) {
  const id = vm.manifest.id
  const load = useCallback(() => listBackups(id), [id])
  const backups = useResource<BackupInfo[]>(load, [], 'vms')
  const action = useRowAction(id, vm.manifest.name, 'vm', run)
  const [restoring, setRestoring] = useState<BackupInfo | null>(null)
  const [removing, setRemoving] = useState<BackupInfo | null>(null)
  const [pruning, setPruning] = useState(false)
  const running = vm.phase === 'running'

  async function backupNow() {
    await action.execute('vm.backup', () => createBackup(id))
  }

  async function restore(asNew: boolean) {
    if (!restoring) return
    const job = await action.execute('vm.backup.restore', () => restoreBackup(id, restoring.timestamp, asNew))
    if (job) setRestoring(null)
  }

  async function remove() {
    if (!removing) return
    const job = await action.execute('vm.backup.delete', () => deleteBackup(id, removing.timestamp))
    if (job) setRemoving(null)
  }

  async function prune() {
    const job = await action.execute('vm.backup.prune', () => pruneBackups(id))
    if (job) setPruning(false)
  }

  return (
    <div className="max-w-3xl space-y-6">
      <ResourceNotice resource={backups} name="backups" />
      <JobNotice error={action.error} />
      <PreferencesGroup
        title="Backups"
        header={
          <Button variant="outline" disabled={action.busy} onClick={backupNow}>
            <StableWidth value={action.action === 'vm.backup' ? 'Backing Up…' : 'Back Up Now'} values={['Back Up Now', 'Backing Up…']} />
          </Button>
        }
      >
        {!backups.loading && backups.data.length === 0 && <Row subtitle="None" />}
        {backups.data.map((backup) => (
          <Row
            key={backup.timestamp}
            title={backupTime(backup)}
            subtitle={`${formatBytes(backup.size_bytes)} · ${plural(backup.disks.length, 'disk')} · ${backupKind(backup)}`}
          >
            <Button disabled={action.busy} onClick={() => setRestoring(backup)}>
              Restore…
            </Button>
            <ActionButton label={`Delete Backup ${backupTime(backup)}`} icon={Trash2} disabled={action.busy} onClick={() => setRemoving(backup)} />
          </Row>
        ))}
      </PreferencesGroup>

      <VMBackupSchedule
        vmID={id}
        busy={action.busy}
        pruning={action.action === 'vm.backup.prune'}
        onPrune={() => setPruning(true)}
      />

      {restoring && (
        <Dialog
          open
          onClose={() => !action.busy && setRestoring(null)}
          title={`Restore ${backupTime(restoring)}`}
          description={running ? 'Shut down to replace this VM; restoring as a new VM keeps it' : 'Replacing overwrites this VM’s disks'}
          footer={
            <>
              <Button variant="outline" disabled={action.busy} onClick={() => setRestoring(null)}>
                Cancel
              </Button>
              <Button variant="destructive" disabled={action.busy || running} onClick={() => restore(false)}>
                Replace This VM
              </Button>
              <Button disabled={action.busy} onClick={() => restore(true)}>
                Restore as New VM
              </Button>
            </>
          }
        />
      )}
      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title={`Delete Backup ${removing ? backupTime(removing) : ''}?`}
        confirmLabel="Delete Backup"
      />
      <AlertDialog
        open={pruning}
        onCancel={() => setPruning(false)}
        onConfirm={prune}
        busy={action.busy}
        destructive
        title="Clean Up Backups?"
        description="Backups not kept by the saved retention rules are permanently deleted"
        confirmLabel="Delete Backups"
      />
    </div>
  )
}
