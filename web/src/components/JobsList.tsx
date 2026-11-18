import { useCallback, useState } from 'react'
import {
  EmptyState,
  SearchInput,
  Pagination,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
  Button,
  Spinner,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from 'maco-ui'
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
  all: 'All',
  active: 'Queued or in progress',
  pending: 'Queued',
  running: 'In progress',
  completed: 'Finished',
  succeeded: 'Completed',
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

  function clearFilters() {
    setSearch('')
    setFilter('all')
    setPage(1)
  }

  function clearSearch() {
    setSearch('')
    setPage(1)
  }

  function changeSearch(event: React.ChangeEvent<HTMLInputElement>) {
    setSearch(event.target.value)
    setPage(1)
  }

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

  const filtered = !!search || filter !== 'all'

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <SearchInput
          aria-label="Search Activity"
          placeholder="Search activity…"
          value={search}
          onChange={changeSearch}
          onClear={clearSearch}
          className="w-full sm:w-72"
        />
        <div className="w-48">
          <Select value={filter} onValueChange={changeFilter}>
            <SelectTrigger aria-label="State">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.keys(filters) as JobFilter[]).map(renderFilter)}
            </SelectContent>
          </Select>
        </div>
        {filtered && (
          <Button variant="ghost" onClick={clearFilters}>
            Clear Filters
          </Button>
        )}
      </div>
      <ResourceNotice resource={history} name="activity" />
      <JobNotice error={resource.error ? `Live updates unavailable: ${resource.error}` : ''} />
      {loading ? (
        <Spinner />
      ) : total === 0 && !history.error ? (
        <EmptyState icon={<ListTodo />} title={filtered ? 'No Results' : 'No Activity'} className="py-10" />
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Action</TableHead>
                <TableHead>Target</TableHead>
                <TableHead>State</TableHead>
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
            unit="jobs"
          />
        </>
      )}
    </div>
  )
}
