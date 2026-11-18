import { useEffect, useState, useRef } from 'react'
import { errorMessage } from '../api'
import { subscribeResourceEvents } from './resourceEvents'

const inFlight = new Map<() => Promise<unknown>, Promise<unknown>>()
function loadShared<T>(load: () => Promise<T>): Promise<T> {
  const existing = inFlight.get(load)
  if (existing) return existing as Promise<T>
  const request = Promise.resolve().then(load).finally(() => { inFlight.delete(load) })
  inFlight.set(load, request)
  return request
}

export function useResource<T>(load: () => Promise<T>, initial: T, domain: string) {
  const [data, setData] = useState(initial)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const refreshRef = useRef<() => void>(() => {})
  useEffect(() => {
    let active = true
    let pending = false
    let dirty = false
    setLoading(true)
    async function refresh() {
      dirty = true
      if (pending) return
      pending = true
      while (active && dirty) {
        dirty = false
        try {
          const result = await loadShared(load)
          if (active) { setData(result); setError('') }
        } catch (error) {
          if (active) setError(errorMessage(error))
        } finally {
          if (active) setLoading(false)
        }
      }
      pending = false
    }
    refreshRef.current = () => { void refresh() }
    const unsubscribe = subscribeResourceEvents(resources => {
      if (resources.includes('all') || resources.includes(domain)) void refresh()
    })
    void refresh()
    return () => { active = false; refreshRef.current = () => {}; unsubscribe() }
  }, [load, domain])
  return { data, error, loading, refresh: () => refreshRef.current() }
}
