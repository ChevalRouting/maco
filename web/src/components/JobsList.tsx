import { useCallback, useState } from 'react'
import {
  EmptyState,
  Input,
  Pagination,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Button,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from 'cheval-ui'
import { ListTodo } from 'lucide-react'
import { listJobs, type Job, type JobsPage } from '../api'
import { type JobsResource } from '../hooks/useJobs'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { JobRow } from './JobRow'
import { useResource } from '../hooks/useResource'

type JobFilter =
  | 'all'
  | 'active'
  | 'pending'
  | 'running'
  | 'completed'
  | 'succeeded'
  | 'failed'

const filters: Record<JobFilter, string> = {
  all: 'All Activity',
  active: 'Active',
  pending: 'Queued',
  running: 'In progress',
  completed: 'Finished (Successful or Failed)',
  succeeded: 'Completed Successfully',
  failed: 'Failed',
}

const PAGE_SIZE = 25

interface JobsListProps {
  resource: JobsResource
  highlightedJob?: string
}

function renderFilter(filter: JobFilter) {
  return (
    <SelectItem key={filter} value={filter}>
      {filters[filter]}
    </SelectItem>
  )
}

export function JobsList({ resource, highlightedJob }: JobsListProps) {
  const [filter, setFilter] = useState<JobFilter>('all')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)

  const load = useCallback(() => listJobs(page, filter, search, highlightedJob), [page, filter, search, highlightedJob])
  const history = useResource<JobsPage | null>(load, null, 'jobs')
  const total = history.data?.total || 0
  const totalPages = history.data?.total_pages || 1
  const current = history.data?.page || 1
  const visible = history.data?.items || []

  function changeFilter(value: string) {
    setFilter(value as JobFilter)
    setPage(1)
  }

  function renderJob(job: Job) {
    return (
      <JobRow key={job.id} job={job} highlighted={job.id === highlightedJob} />
    )
  }

  const loading = history.loading && !history.data

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="w-44">
          <Select value={filter} onValueChange={changeFilter}>
            <SelectTrigger aria-label="Filter activity by status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.keys(filters) as JobFilter[]).map(renderFilter)}
            </SelectContent>
          </Select>
        </div>
        <Input
          aria-label="Search activity"
          placeholder="Search action, target, or activity ID"
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setPage(1)
          }}
          className="w-full sm:w-72"
        />
        <span role="status" className="text-xs text-muted-foreground">
          {loading ? 'Loading…' : `${total} operations`}
        </span>
      </div>
      <ResourceNotice resource={history} name="activity history" />
      <JobNotice
        error={
          resource.error
            ? `Live activity updates are unavailable. History remains available; updates reconnect automatically. ${resource.error}`
            : ''
        }
      />
      {(search || filter !== 'all') && (
        <Button
          variant="ghost"
          onClick={() => {
            setSearch('')
            setFilter('all')
            setPage(1)
          }}
        >
          Clear Filters
        </Button>
      )}
      {loading ? (
        <p role="status" className="text-sm text-muted-foreground">Loading activity…</p>
      ) : total === 0 && !history.error ? (
        <EmptyState
          icon={<ListTodo />}
          title={search || filter !== 'all' ? 'No Matching Activity' : 'No Activity Yet'}
          message={
            search || filter !== 'all'
              ? 'No operations match these filters.'
              : 'Operations you run will appear here.'
          }
          className="py-10"
        />
      ) : (
        <>
          <Table className="table-cards">
            <TableHeader>
              <TableRow>
                <TableHead>Action</TableHead>
                <TableHead>Target</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Submitted</TableHead>
                <TableHead>Finished</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>{visible.map(renderJob)}</TableBody>
          </Table>
          <Pagination
            page={current - 1}
            totalPages={totalPages}
            total={total}
            pageSize={PAGE_SIZE}
            onPage={(value) => setPage(value + 1)}
            unit="operations"
          />
        </>
      )}
    </div>
  )
}
