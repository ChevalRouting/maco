import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Button } from 'cheval-ui'
import { CheckCircle2, ChevronDown, ChevronUp, CircleAlert, Clock, ListTodo, LoaderCircle } from 'lucide-react'
import { type Job } from '../api'
import { activityName, activityState, navOrigin } from '../ux'
import { useJobs } from '../hooks/useJobs'
import { useJobTracking } from '../hooks/useJobTracking'
import { JobsList } from './JobsList'

const STORAGE_KEY = 'maco-jobs-expanded'

function initialExpanded() {
  return localStorage.getItem(STORAGE_KEY) === 'true'
}
function isRunning(job: Job) {
  return job.state === 'running'
}
function isPending(job: Job) {
  return job.state === 'pending'
}


export function JobsPanel() {
  const [expanded, setExpanded] = useState(initialExpanded)
  const resource = useJobs()
  const { notification } = useJobTracking()
  const location = useLocation()
  const { pathname } = location

  const active = resource.data
    .filter(job => job.action !== 'vm.screenshot' && (isRunning(job) || isPending(job)))
    .sort((first, second) => {
      if (first.state !== second.state) return isRunning(first) ? -1 : 1
      return Date.parse(second.started_at || second.created_at) - Date.parse(first.started_at || first.created_at)
    })
  const latest = notification?.job
    ? resource.data.find(job => job.id === notification.job?.id) || notification.job
    : null
  const candidates = latest ? [latest, ...active.filter(job => job.id !== latest.id)] : active
  const previews = candidates.slice(0, notification?.error ? 1 : 2)
  const running = active.filter(isRunning).length
  const pending = active.filter(isPending).length
  const Chevron = expanded ? ChevronDown : ChevronUp

  function toggle() {
    const next = !expanded
    setExpanded(next)
    localStorage.setItem(STORAGE_KEY, String(next))
  }

  if (pathname === '/jobs') return null

  return (
    <section
      aria-label="Activity panel"
      className="shrink-0 border-t border-border bg-background"
    >
      <div className="flex min-h-11 flex-wrap items-center justify-between gap-2 px-3">
        <Button
          variant="ghost"
          onClick={toggle}
          aria-expanded={expanded}
          aria-controls="bottom-jobs"
          className="shrink-0 gap-2"
        >
          <ListTodo className="h-4 w-4 shrink-0" />
          <span>Activity</span>
          <span
            role="status"
            className="hidden text-xs text-muted-foreground sm:inline"
          >
            {resource.error
              ? 'Updates unavailable'
              : resource.loading
                ? 'Loading…'
                : `${running} in progress · ${pending} queued`}
          </span>
          <Chevron className="h-4 w-4 shrink-0" />
        </Button>
        <div role="status" aria-live="polite" className="flex min-w-0 flex-1 items-center gap-2 px-2 py-1">
          {notification?.error && (
            <span
              key={notification.sequence}
              title={notification.error}
              className="jobs-preview-bump flex min-w-0 flex-1 items-center gap-2 rounded-md bg-destructive/10 px-2 py-1 text-xs text-destructive"
            >
              <CircleAlert aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
              <span className="min-w-0 break-words">{notification.error}</span>
            </span>
          )}
          {previews.map((job, index) => {
            const Icon = isRunning(job) ? LoaderCircle : isPending(job) ? Clock : job.state === 'failed' ? CircleAlert : CheckCircle2
            const status = activityState(job)
            const updated = notification?.job?.id === job.id
            const color = job.state === 'failed' ? 'text-destructive' : job.state === 'succeeded' ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'
            const label = `${activityName(job)} · ${job.target}`
            const primary = index === 0 && !notification?.error
            return (
              <Link
                key={updated ? `${job.id}:${notification.sequence}` : job.id}
                to={`/jobs/${encodeURIComponent(job.id)}`}
                state={{ from: navOrigin(location) }}
                title={`${label} · ${status}${job.error ? ` · ${job.error}` : ''}`}
                className={`min-w-0 items-center gap-2 rounded-md bg-muted px-2 py-1 text-xs hover:bg-muted/80 ${updated ? 'jobs-preview-bump' : ''} ${primary ? 'flex flex-1' : 'hidden max-w-72 md:flex'}`}
              >
                <Icon aria-hidden="true" className={`h-3.5 w-3.5 shrink-0 ${color} ${isRunning(job) ? 'animate-spin motion-reduce:animate-none' : ''}`} />
                <span className="truncate">{label}</span>
                {!isRunning(job) && !isPending(job) && <span className={`shrink-0 ${color}`}>{status}</span>}
                {(isRunning(job) || isPending(job)) && <span className="sr-only">{status}</span>}
              </Link>
            )
          })}
          {candidates.length > 1 && (
            <span className="shrink-0 text-xs text-muted-foreground md:hidden">
              +{candidates.length - 1}
            </span>
          )}
          {candidates.length > previews.length && (
            <span className="hidden shrink-0 text-xs text-muted-foreground md:inline">
              +{candidates.length - previews.length}
            </span>
          )}
          {!previews.length && !notification?.error && (
            <span className="truncate text-xs text-muted-foreground">
              {resource.error
                ? 'Updates unavailable'
                : resource.loading
                  ? 'Loading…'
                  : 'No active operations'}
            </span>
          )}
        </div>
        <Link
          to="/jobs"
          className="shrink-0 text-xs text-primary hover:underline"
        >
          All Activity
        </Link>
      </div>
      <div
        id="bottom-jobs"
        hidden={!expanded}
        className="max-h-[min(24rem,40dvh)] overflow-auto border-t border-border p-4"
      >
        <JobsList
          resource={resource}
        />
      </div>
    </section>
  )
}
