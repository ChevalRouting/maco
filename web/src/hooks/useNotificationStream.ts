import { useEffect, useState } from 'react'
import { getToken, clearToken, type Job } from '../api'
import { notifyResources } from './resourceEvents'
import { type JobsResource } from './useJobs'

interface NotificationEvent {
  type: 'snapshot' | 'update' | 'invalidate'
  resources?: string[]
  jobs?: Job[]
  job?: Job
}

const initialResource: JobsResource = { data: [], loading: true, error: '' }

function applyEvent(current: JobsResource, event: NotificationEvent): JobsResource {
  if (event.type === 'snapshot') {
    return { data: event.jobs || [], loading: false, error: '' }
  }

  if (!event.job) return current

  const job = event.job
  const others = current.data.filter((existing) => existing.id !== job.id)
  return { data: [job, ...others], loading: false, error: '' }
}

function disconnected(current: JobsResource): JobsResource {
  return {
    ...current,
    loading: false,
    error: 'Notification stream disconnected. Reconnecting…',
  }
}

const hydrationTimeout = 10000

class EventsConnection {
  private socket?: WebSocket
  private timer?: ReturnType<typeof setTimeout>
  private hydration?: ReturnType<typeof setTimeout>
  private active = true
  private delay = 1000

  constructor(
    private update: (change: (current: JobsResource) => JobsResource) => void,
  ) {
    this.connect()
  }

  private connect = () => {
    if (!this.active) return

    const url = new URL('/api/events', window.location.href)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    this.socket = new WebSocket(url)
    this.socket.onopen = this.authenticate
    this.socket.onmessage = this.receive
    this.socket.onclose = this.closed
    this.socket.onerror = this.failed
    clearTimeout(this.hydration)
    this.hydration = setTimeout(this.hydrationExpired, hydrationTimeout)
  }

  private hydrationExpired = () => {
    if (!this.active) return
    this.update((current) => (current.loading ? { ...current, loading: false } : current))
  }

  private authenticate = () => {
    this.socket?.send(JSON.stringify({ token: getToken() }))
  }

  private receive = (message: MessageEvent<string>) => {
    if (!this.active) return
    clearTimeout(this.hydration)

    try {
      const event = JSON.parse(message.data) as NotificationEvent
      if (event.type === 'invalidate') {
        this.delay = 1000
        notifyResources(event.resources || [])
        return
      }
      if (event.type !== 'snapshot' && event.type !== 'update') return
      if (event.type === 'snapshot') notifyResources(['all'])
      if (event.job) notifyResources(['jobs'])

      this.delay = 1000
      this.update((current) => applyEvent(current, event))
    } catch {
      this.socket?.close()
    }
  }

  private failed = () => {
    if (this.active) this.update(disconnected)
  }

  private closed = (event: CloseEvent) => {
    if (!this.active) return

    if (event.code === 1008) {
      this.active = false
      clearToken()
      window.location.assign('/login')
      return
    }

    this.update(disconnected)
    this.timer = setTimeout(this.connect, this.delay)
    this.delay = Math.min(this.delay * 2, 15000)
  }

  dispose() {
    this.active = false
    clearTimeout(this.timer)
    clearTimeout(this.hydration)
    this.socket?.close()
  }
}

export function useNotificationStream() {
  const [resource, setResource] = useState(initialResource)

  function subscribe() {
    const connection = new EventsConnection(setResource)
    return () => connection.dispose()
  }

  useEffect(subscribe, [])
  return resource
}
