import { useState } from 'react'
import { errorMessage, type Job } from '../api'
import { useJobs } from './useJobs'

export function useRowAction(
  id: string,
  name: string,
  scope: 'vm' | 'network' | 'image',
  run: (action: () => Promise<Job>) => Promise<Job | null>,
) {
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState('')
  const [submitted, setSubmitted] = useState<Job | null>(null)
  const jobs = useJobs()
  const tracked = jobs.data.find((job) => job.id === submitted?.id) || submitted
  const active = jobs.data.find(
    (job) =>
      job.action.startsWith(`${scope}.`) &&
      job.action !== 'vm.screenshot' &&
      (job.target === id || job.target === name) &&
      (job.state === 'pending' || job.state === 'running'),
  )
  const job = active || tracked
  const busy =
    !!submitting || job?.state === 'pending' || job?.state === 'running'

  async function execute(action: string, request: () => Promise<Job>) {
    setSubmitting(action)
    setError('')
    try {
      const result = await run(async () => {
        try { return await request() } catch (error) { setError(errorMessage(error)); throw error }
      })
      if (result) setSubmitted(result)
      return result
    } finally {
      setSubmitting('')
    }
  }

  return { busy, error, action: submitting || (busy ? job?.action : ''), job, execute }
}
