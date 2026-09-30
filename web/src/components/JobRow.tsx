import { JobTarget } from './JobTarget'
import { JobStatusBadge } from './StatusBadge'
import { Link, useLocation } from 'react-router-dom'
import { TableRow, TableCell } from 'cheval-ui'
import { type Job } from '../api'
import { activityName, unfinishedTime, navOrigin } from '../ux'

interface JobRowProps {
  job: Job
  highlighted?: boolean
}

export function JobRow({ job, highlighted }: JobRowProps) {
  const location = useLocation()
  return (
    <TableRow
      aria-selected={highlighted || undefined}
      className={highlighted ? 'bg-primary/10 hover:bg-primary/15' : undefined}
    >
      <TableCell data-label="Action">
        <Link className="text-primary hover:underline" to={`/jobs/${job.id}`} state={{ from: navOrigin(location) }}>
          {activityName(job)}
        </Link>
      </TableCell>
      <TableCell data-label="Target">
        <JobTarget job={job} />
      </TableCell>
      <TableCell data-label="Status">
        <JobStatusBadge state={job.state} job={job} />
      </TableCell>
      <TableCell data-label="Submitted">{new Date(job.created_at).toLocaleString()}</TableCell>
      <TableCell data-label="Finished">
        {job.finished_at
          ? new Date(job.finished_at).toLocaleString()
          : unfinishedTime(job)}
      </TableCell>
    </TableRow>
  )
}
