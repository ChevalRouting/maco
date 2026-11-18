import { StatusChip, type StatusIntent } from 'maco-ui'
import { type Job } from '../api'
import { activityState } from '../ux'
import { StableWidth } from './StableWidth'

const jobIntents: Record<Job['state'], StatusIntent> = {
  pending: 'warning',
  running: 'info',
  succeeded: 'success',
  failed: 'danger',
}

const jobLabels = ['Queued', 'In Progress', 'Completed', 'Failed', 'Shutdown Requested']

interface JobStateChipProps {
  job: Job
  align?: 'start' | 'end'
}

export function JobStateChip({ job, align = 'start' }: JobStateChipProps) {
  const label = activityState(job)
  return (
    <StableWidth
      align={align}
      value={label}
      values={jobLabels}
      render={(option) => <StatusChip intent={option === label ? jobIntents[job.state] : 'neutral'} label={option} className="whitespace-nowrap" />}
    />
  )
}

const vmLabels: Record<string, string> = {
  running: 'Running',
  stopped: 'Stopped',
}

interface VMStateChipProps {
  phase: string
  align?: 'start' | 'end'
}

export function VMStateChip({ phase, align = 'start' }: VMStateChipProps) {
  return (
    <StableWidth
      align={align}
      value={phase}
      values={Object.keys(vmLabels)}
      render={(option) => (
        <StatusChip intent={option === 'running' ? 'success' : 'neutral'} label={vmLabels[option] ?? option} className="whitespace-nowrap" />
      )}
    />
  )
}
