import { JobTarget } from '../components/JobTarget'
import { JobStatusBadge } from '../components/StatusBadge'
import { useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader, PreferencesGroup, Row, SectionLabel } from 'cheval-ui'
import { getJob, type Job } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobs } from '../hooks/useJobs'
import { BackLink } from '../components/BackLink'
import { activityName, unfinishedTime } from '../ux'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'

export default function JobDetails() {
  const { id = '' } = useParams()
  const load = useCallback(() => getJob(id), [id])
  const resource = useResource<Job | null>(load, null, 'jobs')
  const stream = useJobs()
  const streamedJob = stream.data.find((job) => job.id === id)
  const job = streamedJob
    ? { ...streamedJob, logs: resource.data?.logs || [] }
    : resource.data

  return (
    <div className="space-y-6">
      <BackLink to="/jobs">Back to Activity</BackLink>
      <PageHeader title={job ? activityName(job) : 'Activity Details'} />
      <ResourceNotice resource={resource} name="activity details" />
      <JobNotice error={stream.error} />
      {!resource.loading && !resource.error && !job && (
        <p className="text-sm text-muted-foreground">
          This activity record is no longer available. Return to Activity to view recent operations.
        </p>
      )}
      {job && (
        <>
          <PreferencesGroup title="Operation">
            <Row title="Status">
              <JobStatusBadge state={job.state} job={job} />
            </Row>
            <Row title="Target">
              <JobTarget job={job} />
            </Row>
            <Row title="Submitted">
              {new Date(job.created_at).toLocaleString()}
            </Row>
            <Row title="Started">
              {job.started_at
                ? new Date(job.started_at).toLocaleString()
                : job.state === 'pending' ? 'Not Started' : 'Not Recorded'}
            </Row>
            <Row title="Finished">
              {job.finished_at
                ? new Date(job.finished_at).toLocaleString()
                : unfinishedTime(job)}
            </Row>
            {job.error && (
              <Row
                title="Error"
                subtitle={<span className="break-words text-destructive">{job.error}</span>}
              />
            )}
          </PreferencesGroup>
          <div className="space-y-2">
            <SectionLabel>Logs</SectionLabel>
            <pre
              aria-label="Job logs"
              className="max-h-[32rem] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-card p-4 text-xs font-mono"
            >
              {job.logs.join('\n') ||
                (job.state === 'succeeded' || job.state === 'failed'
                  ? 'No log output was recorded.'
                  : 'No log output yet.')}
            </pre>
          </div>
        </>
      )}
    </div>
  )
}
