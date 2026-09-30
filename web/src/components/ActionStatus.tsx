import { Link, useLocation } from 'react-router-dom'
import { NoticeBanner } from 'cheval-ui'
import { type Job } from '../api'
import { activityName, activityState, navOrigin } from '../ux'

interface ActionStatusProps {
  job?: Job | null
  error?: string
  hideProgress?: boolean
}

export function ActionStatus({ job, error, hideProgress }: ActionStatusProps) {
  const location = useLocation()
  if (!job && !error) return null
  const failure = error || (job?.state === 'failed' ? job.error : '')
  const showLink = job && !hideProgress
  if (!showLink && !failure) return null

  return (
    <div role="status" className="min-w-0 space-y-2 text-sm">
      {showLink && (
        <Link
          to={`/jobs/${encodeURIComponent(job.id)}`}
          state={{ from: navOrigin(location) }}
          className="inline-block whitespace-normal break-words text-primary hover:underline"
        >
          {activityName(job)} · {activityState(job)} · View Details
        </Link>
      )}
      {failure && (
        <NoticeBanner intent="danger">
          <span className="break-words">{failure}</span>
        </NoticeBanner>
      )}
    </div>
  )
}
