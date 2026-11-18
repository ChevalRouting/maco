import { useEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { Sheet, cn } from 'maco-ui'
import { Check, ListTodo } from 'lucide-react'
import { type Job } from '../api'
import { useJobs } from '../hooks/useJobs'
import { useJobTracking, type JobNotification } from '../hooks/useJobTracking'
import { useJobToasts } from '../hooks/useJobToasts'
import { JobsList } from './JobsList'

const celebrated = new WeakSet<JobNotification>()

function isActive(job: Job) {
  return job.action !== 'vm.screenshot' && (job.state === 'running' || job.state === 'pending')
}

export function MobileJobsIndicator() {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const resource = useJobs()
  const { notification } = useJobTracking()

  const [finished, setFinished] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout>>()
  useJobToasts()

  useEffect(() => setOpen(false), [pathname])

  useEffect(() => {
    if (!notification || notification.job?.state !== 'succeeded' || celebrated.has(notification)) return
    celebrated.add(notification)
    setFinished(true)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => setFinished(false), 2500)
  }, [notification])

  useEffect(() => () => clearTimeout(timer.current), [])

  const active = resource.data.filter(isActive).length
  const failed = Boolean(notification?.error) || notification?.job?.state === 'failed'
  const label = failed ? 'Activity, Last Job Failed' : active > 0 ? `Activity, ${active} Active` : 'Activity'

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={label}
        title={label}
        className="fixed right-2 top-[calc(var(--app-top-inset,0px)+0.125rem)] z-30 flex h-10 w-10 items-center justify-center rounded-md text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground md:hidden"
      >
        {active > 0 && (
          <span aria-hidden="true" className="absolute inset-1 animate-spin rounded-full border-2 border-info/25 border-t-info" />
        )}
        {finished && !active && !failed ? (
          <Check aria-hidden="true" className="h-5 w-5 animate-in zoom-in-50 text-success duration-300" />
        ) : (
          <ListTodo className={cn('h-5 w-5', active > 0 && 'scale-75 transition-transform')} />
        )}
        {active > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-info px-1 text-[10px] font-semibold leading-none text-info-foreground ring-2 ring-sidebar animate-in zoom-in-50 duration-200">
            {active}
          </span>
        )}
        {failed && !active && (
          <span className="absolute right-1.5 top-1.5 h-2.5 w-2.5 rounded-full bg-danger ring-2 ring-sidebar" />
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
