import { useJobTracking } from './useJobTracking'
import { useState } from 'react'
import { errorMessage, type Job } from '../api'

export function useJobAction() {
  const { trackJob, reportError } = useJobTracking()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [job, setJob] = useState<Job | null>(null)

  async function run(action: () => Promise<Job>) {
    setBusy(true)
    setError('')
    try {
      const result = await action()
      setJob(result)
      trackJob(result)
      return result
    } catch (error) {
      const message = errorMessage(error)
      setError(message)
      reportError(message)
      return null
    } finally {
      setBusy(false)
    }
  }

  return { busy, error, job, run, track: trackJob }
}
