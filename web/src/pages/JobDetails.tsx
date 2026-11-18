import { useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { EmptyState, PageHeader, PreferencesGroup, Row } from 'maco-ui'
import { ListTodo } from 'lucide-react'
import { JobTarget } from '../components/JobTarget'
import { JobStateChip } from '../components/StateChips'
import { getJob, type Job } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobs } from '../hooks/useJobs'
import { usePageBack } from '../components/usePageBack'
import { activityName, unfinishedTime } from '../ux'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'

function startedText(job: Job): string {
  if (job.started_at) return new Date(job.started_at).toLocaleString()
  return job.state === 'pending' ? 'Not started' : 'Not recorded'
}

function emptyLog(job: Job): string {
  return job.state === 'succeeded' || job.state === 'failed' ? 'No output' : 'No output yet'
}

export default function JobDetails() {
  const { id = '' } = useParams()
  const back = usePageBack('/jobs', 'Activity')
  const load = useCallback(() => getJob(id), [id])
  const resource = useResource<Job | null>(load, null, 'jobs')
  const stream = useJobs()
  const streamedJob = stream.data.find((job) => job.id === id)
  const job = streamedJob ? { ...streamedJob, logs: resource.data?.logs || [] } : resource.data

  return (
    <div className="space-y-6">
      <PageHeader title={job ? activityName(job) : 'Activity'} back={back} />
      <ResourceNotice resource={resource} name="activity" />
      <JobNotice error={stream.error} />
      {!resource.loading && !resource.error && !job && <EmptyState icon={<ListTodo />} title="Job Not Found" />}
      {job && (
        <>
          <div className="mx-auto max-w-3xl">
            <PreferencesGroup>
              <Row title="State">
                <JobStateChip job={job} align="end" />
              </Row>
              <Row title="Target">
                <JobTarget job={job} />
              </Row>
              <Row title="Submitted">
                <span className="text-sm tabular-nums">{new Date(job.created_at).toLocaleString()}</span>
              </Row>
              <Row title="Started">
                <span className="text-sm tabular-nums">{startedText(job)}</span>
              </Row>
              <Row title="Finished">
                <span className="text-sm tabular-nums">{job.finished_at ? new Date(job.finished_at).toLocaleString() : unfinishedTime(job)}</span>
              </Row>
              {job.error && <Row title="Error" subtitle={<span className="break-words text-danger">{job.error}</span>} />}
            </PreferencesGroup>
          </div>
          <section className="space-y-2" aria-labelledby="job-log-title">
            <h2 id="job-log-title" className="px-1 text-sm font-semibold">Log</h2>
            <pre className="max-h-[32rem] overflow-auto whitespace-pre-wrap break-words rounded-xl bg-card p-4 font-mono text-xs [box-shadow:var(--card-shadow)]">
              {job.logs.join('\n') || emptyLog(job)}
            </pre>
          </section>
        </>
      )}
    </div>
  )
}
