import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Sheet, cn } from 'cheval-ui'
import { ListTodo } from 'lucide-react'
import { type Job } from '../api'
import { useJobs } from '../hooks/useJobs'
import { useJobTracking } from '../hooks/useJobTracking'
import { JobsList } from './JobsList'

function isRunning(job: Job) {
  return job.state === 'running'
}
function isPending(job: Job) {
  return job.state === 'pending'
}

export function MobileJobsIndicator() {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const resource = useJobs()
  const { notification } = useJobTracking()

  useEffect(() => setOpen(false), [pathname])

  const active = resource.data.filter(
    (job) => job.action !== 'vm.screenshot' && (isRunning(job) || isPending(job)),
  )
  const running = active.filter(isRunning).length
  const pending = active.filter(isPending).length
  const failed = Boolean(notification?.error) || notification?.job?.state === 'failed'

  const intent = failed ? 'error' : active.length > 0 ? 'active' : 'idle'
  const label =
    intent === 'error'
      ? 'Activity: last operation failed'
      : intent === 'active'
        ? `Activity: ${running} in progress, ${pending} queued`
        : 'Activity'

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={label}
        title={label}
        className="md:hidden fixed right-2 top-[calc(var(--app-top-inset,0px)+0.375rem)] z-30 flex h-9 w-9 items-center justify-center rounded-md text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
      >
        <ListTodo className="h-5 w-5" />
        {intent !== 'idle' && (
          <span className="absolute right-1 top-1 flex h-2.5 w-2.5">
            <span
              className={cn(
                'absolute inline-flex h-full w-full animate-ping rounded-full opacity-75 motion-reduce:animate-none',
                intent === 'error' ? 'bg-destructive' : 'bg-green-500',
              )}
            />
            <span
              className={cn(
                'relative inline-flex h-2.5 w-2.5 rounded-full ring-2 ring-sidebar',
                intent === 'error' ? 'bg-destructive' : 'bg-green-500',
              )}
            />
          </span>
        )}
      </button>
      {open && (
        <Sheet open={open} onClose={() => setOpen(false)} title="Activity">
          <JobsList resource={resource} />
        </Sheet>
      )}
    </>
  )
}
