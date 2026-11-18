import { createContext, useContext } from 'react'
import type { Job } from '../api'

export interface JobNotification {
  sequence: number
  job?: Job
  error?: string
}

interface JobTracking {
  trackJob: (job: Job) => void
  reportError: (message: string) => void
  notification: JobNotification | null
}

export const JobTrackingContext = createContext<JobTracking | null>(null)

export function useJobTracking() {
  const tracking = useContext(JobTrackingContext)
  if (!tracking) throw new Error('JobsProvider is required')
  return tracking
}
