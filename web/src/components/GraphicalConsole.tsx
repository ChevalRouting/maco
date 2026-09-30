import { useEffect, useRef, useState } from 'react'
import { Button } from 'cheval-ui'
import RFB from '@novnc/novnc'
import { getToken } from '../api'

interface GraphicalConsoleProps {
  id: string
  autoConnect?: boolean
}

class DisplaySession {
  private socket: WebSocket
  private display?: RFB
  private active = true

  constructor(
    private element: HTMLElement,
    id: string,
    private status: (value: string, live: boolean) => void,
  ) {
    const url = new URL(
      `/api/vms/${encodeURIComponent(id)}/display`,
      window.location.href,
    )
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    this.socket = new WebSocket(url)
    this.socket.binaryType = 'arraybuffer'
    this.socket.onopen = this.authenticate
    this.socket.onmessage = this.ready
    this.socket.onclose = this.closed
    this.socket.onerror = this.failed
  }

  private authenticate = () => {
    this.socket.send(JSON.stringify({ token: getToken() }))
  }

  private ready = (event: MessageEvent) => {
    if (!this.active || event.data !== '{"type":"ready"}') return

    this.display = new RFB(this.element, this.socket)
    this.display.clipViewport = false
    this.display.scaleViewport = true
    this.display.resizeSession = false
    this.display.addEventListener('connect', this.connected)
    this.display.addEventListener('disconnect', this.disconnected)
    this.display.addEventListener('securityfailure', this.failed)
  }

  private connected = () => {
    if (!this.active) return
    this.status('Connected', true)
    this.display?.focus()
  }

  private disconnected = () => {
    if (this.active) this.status('Disconnected. Connect to try again.', false)
  }

  private closed = (event: CloseEvent) => {
    if (this.active) this.status(event.reason || 'Disconnected', false)
  }

  private failed = () => {
    if (this.active) this.status('Graphical console connection failed', false)
  }

  dispose() {
    this.active = false
    this.display?.disconnect()
    this.socket.close()
  }
}

export function GraphicalConsole({
  id,
  autoConnect = false,
}: GraphicalConsoleProps) {
  const releaseButton = useRef<HTMLButtonElement>(null)
  const container = useRef<HTMLDivElement>(null)
  const [connected, setConnected] = useState(autoConnect)
  const [status, setStatus] = useState('Disconnected')

  function connectDisplay() {
    if (!connected || !container.current) return

    setStatus('Connecting…')
    const session = new DisplaySession(container.current, id, updateStatus)
    return () => session.dispose()
  }

  useEffect(connectDisplay, [id, connected])

  function updateStatus(value: string, live: boolean) {
    setStatus(value)
    if (!live) setConnected(false)
  }

  function connect() {
    setConnected(true)
  }

  function disconnect() {
    setConnected(false)
    setStatus('Disconnected')
  }

  return (
    <section className="min-w-0 space-y-3" onKeyDownCapture={event => {
      if (event.key === 'Escape' && event.ctrlKey && event.altKey) { event.preventDefault(); event.stopPropagation(); releaseButton.current?.focus() }
    }}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="font-semibold">Graphical console</h2>
          <p role="status" className="text-sm text-muted-foreground">
            {status}
          </p>
        </div>
        <div className="flex gap-2">
          <Button ref={releaseButton} variant="outline" onClick={connected ? disconnect : connect}>
            {connected ? 'Disconnect' : 'Connect'}
          </Button>
        </div>
      </div>
      <div className="rounded-xl border border-border bg-background p-1">
        <div
          data-console-input
          ref={container}
          className="h-[min(32rem,55dvh)] min-h-0 w-full min-w-0 overflow-hidden"
          aria-label="VM graphical display"
        />
      </div>
    </section>
  )
}
