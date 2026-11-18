import { createContext, useContext } from 'react'
import { type Job } from '../api'

export interface JobsResource {
  data: Job[]
  error: string
  loading: boolean
}

export const JobsContext = createContext<JobsResource | null>(null)

export function useJobs() {
  const resource = useContext(JobsContext)
  if (!resource) {
    throw new Error('JobsProvider is required')
  }

  return resource
}
