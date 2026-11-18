import { useEffect, useRef, useState, type ReactNode } from 'react'
import { type Job } from '../api'
import { JobTrackingContext, type JobNotification } from '../hooks/useJobTracking'
import { JobsContext } from '../hooks/useJobs'
import { useNotificationStream } from '../hooks/useNotificationStream'

interface JobsProviderProps {
  children: ReactNode
}

export function JobsProvider({ children }: JobsProviderProps) {
  const resource = useNotificationStream()
  const [submitted, setSubmitted] = useState<Job[]>([])
  const [notification, setNotification] = useState<JobNotification | null>(null)
  const states = useRef(new Map<string, Job['state']>())

  function trackJob(job: Job) {
    if (job.action === 'vm.screenshot') return
    states.current.set(job.id, job.state)
    setSubmitted(current => [job, ...current.filter(item => item.id !== job.id)])
    setNotification(current => ({ sequence: (current?.sequence || 0) + 1, job }))
  }

  function reportError(error: string) {
    setNotification(current => ({ sequence: (current?.sequence || 0) + 1, error }))
  }

  useEffect(() => {
    const completed: string[] = []
    for (const original of [...submitted].reverse()) {
      const job = resource.data.find(item => item.id === original.id) || original
      if (states.current.get(job.id) !== job.state) {
        states.current.set(job.id, job.state)
        setNotification(current => ({ sequence: (current?.sequence || 0) + 1, job }))
      }
      if (job.state === 'succeeded' || job.state === 'failed') completed.push(job.id)
    }
    if (completed.length) {
      for (const id of completed) states.current.delete(id)
      setSubmitted(current => current.filter(job => !completed.includes(job.id)))
    }
  }, [resource.data, submitted])

  const data = [...submitted.filter(job => !resource.data.some(item => item.id === job.id)), ...resource.data]

  return (
    <JobTrackingContext.Provider value={{ trackJob, reportError, notification }}>
      <JobsContext.Provider value={{ ...resource, data }}>
        {children}
      </JobsContext.Provider>
    </JobTrackingContext.Provider>
  )
}
