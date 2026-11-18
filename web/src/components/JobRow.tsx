import { Link, useLocation } from 'react-router-dom'
import { TableRow, TableCell, cn } from 'maco-ui'
import { useRowLink } from '../hooks/useActivate'
import { JobTarget } from './JobTarget'
import { JobStateChip } from './StateChips'
import { type Job } from '../api'
import { activityName, unfinishedTime, navOrigin } from '../ux'

interface JobRowProps {
  job: Job
  highlighted?: boolean
}

export function JobRow({ job, highlighted }: JobRowProps) {
  const location = useLocation()
  const link = useRowLink(`/jobs/${job.id}`, { from: navOrigin(location) })

  return (
    <TableRow aria-selected={highlighted || undefined} onClick={link.onClick} className={cn(link.className, highlighted && 'bg-primary/10')}>
      <TableCell data-title>
        <Link className="text-primary hover:underline" to={`/jobs/${job.id}`} state={{ from: navOrigin(location) }}>
          {activityName(job)}
        </Link>
      </TableCell>
      <TableCell>
        <JobTarget job={job} />
      </TableCell>
      <TableCell>
        <JobStateChip job={job} />
      </TableCell>
      <TableCell className="tabular-nums">{new Date(job.created_at).toLocaleString()}</TableCell>
      <TableCell className="tabular-nums">{job.finished_at ? new Date(job.finished_at).toLocaleString() : unfinishedTime(job)}</TableCell>
    </TableRow>
  )
}
