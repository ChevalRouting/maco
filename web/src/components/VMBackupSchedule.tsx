import { useCallback, useEffect, useState } from 'react'
import { Button, NoticeBanner, PreferencesGroup, SwitchRow } from 'maco-ui'
import {
  getBackupSchedule,
  setBackupSchedule,
  errorMessage,
  type BackupRetention,
  type BackupSchedule,
} from '../api'
import { toast } from 'sonner'
import { useResource } from '../hooks/useResource'
import { IntegerEntryRow } from './IntegerEntryRow'
import { StableWidth } from './StableWidth'

interface VMBackupScheduleProps {
  vmID: string
  busy: boolean
  pruning: boolean
  onPrune: () => void
}

interface RetentionRow {
  field: keyof BackupRetention
  title: string
  help: string
}

const retentionRows: RetentionRow[] = [
  { field: 'keep_last', title: 'Last', help: 'Newest backups' },
  { field: 'keep_daily', title: 'Daily', help: 'Newest of each day' },
  { field: 'keep_weekly', title: 'Weekly', help: 'Newest of each week' },
  { field: 'keep_monthly', title: 'Monthly', help: 'Newest of each month' },
  { field: 'keep_yearly', title: 'Yearly', help: 'Newest of each year' },
]

const noRetention: BackupRetention = { keep_last: 0, keep_daily: 0, keep_weekly: 0, keep_monthly: 0, keep_yearly: 0 }

function scheduleRetention(schedule: BackupSchedule): BackupRetention {
  return {
    keep_last: schedule.keep_last,
    keep_daily: schedule.keep_daily,
    keep_weekly: schedule.keep_weekly,
    keep_monthly: schedule.keep_monthly,
    keep_yearly: schedule.keep_yearly,
  }
}

function sameRetention(a: BackupRetention, b: BackupRetention): boolean {
  return retentionRows.every(({ field }) => a[field] === b[field])
}

function hasRetention(retention: BackupRetention): boolean {
  return retentionRows.some(({ field }) => retention[field] > 0)
}

export function VMBackupSchedule({ vmID, busy, pruning, onPrune }: VMBackupScheduleProps) {
  const load = useCallback(() => getBackupSchedule(vmID), [vmID])
  const schedule = useResource<BackupSchedule | null>(load, null, 'vms')
  const [enabled, setEnabled] = useState(false)
  const [interval, setInterval] = useState(24)
  const [retention, setRetention] = useState<BackupRetention>(noRetention)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const data = schedule.data
    if (!data) return
    setEnabled(data.enabled)
    setInterval(data.interval_hours || 24)
    setRetention(scheduleRetention(data))
  }, [schedule.data])

  const valid = !enabled || interval >= 1
  const retentionDirty = !!schedule.data && !sameRetention(retention, scheduleRetention(schedule.data))
  const dirty =
    !!schedule.data &&
    (enabled !== schedule.data.enabled ||
      interval !== (schedule.data.interval_hours || 24) ||
      retentionDirty)

  function setRule(field: keyof BackupRetention, value: number) {
    setRetention((current) => ({ ...current, [field]: value }))
  }

  async function save() {
    if (saving || !valid) return
    setSaving(true)
    setError('')
    try {
      await setBackupSchedule(vmID, { enabled, interval_hours: interval, ...retention })
      toast.success('Schedule saved')
      schedule.refresh()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  const lastRun = schedule.data?.last_run_at
    ? new Date(schedule.data.last_run_at * 1000).toLocaleString()
    : 'never'

  return (
    <div className="space-y-3">
      {schedule.error && <NoticeBanner intent="danger">{schedule.error}</NoticeBanner>}
      {error && <NoticeBanner intent="danger">{error}</NoticeBanner>}
      <fieldset disabled={saving} className="space-y-6">
        <PreferencesGroup title="Schedule" description={`Last backup: ${lastRun}`}>
          <SwitchRow title="Back Up Automatically" checked={enabled} onCheckedChange={setEnabled} />
          <IntegerEntryRow id="backup-interval" title="Interval (hours)" min={1} value={interval} disabled={!enabled} onValueChange={setInterval} />
        </PreferencesGroup>
        <PreferencesGroup
          title="Retention"
          description="Applied after each scheduled backup, 0 disables a rule"
          header={
            <Button variant="outline" disabled={busy || saving || retentionDirty || !hasRetention(retention)} onClick={onPrune}>
              <StableWidth value={pruning ? 'Cleaning Up…' : 'Clean Up Now'} values={['Clean Up Now', 'Cleaning Up…']} />
            </Button>
          }
        >
          {retentionRows.map((row) => (
            <IntegerEntryRow
              key={row.field}
              id={`backup-${row.field}`}
              title={row.title}
              min={0}
              value={retention[row.field]}
              help={row.help}
              onValueChange={(value) => setRule(row.field, value)}
            />
          ))}
        </PreferencesGroup>
      </fieldset>
      <div className="flex justify-end">
        <Button variant="suggested" disabled={saving || !valid || !dirty} onClick={save}>
          <StableWidth value={saving ? 'Saving…' : 'Save'} values={['Save', 'Saving…']} />
        </Button>
      </div>
    </div>
  )
}
