import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { type Job } from '../api'
import { activityName, activityState } from '../ux'
import { useJobTracking, type JobNotification } from './useJobTracking'

const compact = '(max-width: 767px)'
const shown = new WeakSet<JobNotification>()

function showJob(job: Job, open: () => void) {
  const title = activityName(job)
  const action = { label: 'Details', onClick: open }
  const classNames = { content: 'min-w-0 flex-1', description: 'line-clamp-3 [overflow-wrap:anywhere]' }
  if (job.state === 'succeeded') {
    toast.success(title, { id: job.id, description: activityState(job), action, classNames, duration: 3000 })
  } else if (job.state === 'failed') {
    toast.error(title, { id: job.id, description: job.error || activityState(job), action, classNames, duration: 8000 })
  } else {
    toast.loading(title, { id: job.id, description: activityState(job), action, classNames, duration: Infinity })
  }
}

export function useJobToasts() {
  const navigate = useNavigate()
  const { notification } = useJobTracking()

  useEffect(() => {
    if (!notification || shown.has(notification) || !window.matchMedia(compact).matches) return
    shown.add(notification)
    const { job, error } = notification
    if (job) showJob(job, () => navigate(`/jobs/${job.id}`))
    else if (error) toast.error(error, { duration: 8000 })
  }, [notification, navigate])
}
