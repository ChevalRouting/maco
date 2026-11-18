import { useEffect, useRef, useState } from 'react'
import { Button } from 'maco-ui'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { getToken } from '../api'
import { terminalTheme } from '../lib/terminalTheme'
import '@xterm/xterm/css/xterm.css'

interface SerialConsoleProps {
  id: string
  autoConnect?: boolean
}

class ConsoleSession {
  private terminal = new Terminal({
    cursorBlink: true,
    convertEol: false,
    fontSize: 13,
    theme: terminalTheme(),
  })
  private fit = new FitAddon()
  private socket: WebSocket
  private observer: ResizeObserver
  private active = true
  private frame = 0
  private replayed = false
  private replaying = false

  constructor(
    element: HTMLElement,
    id: string,
    private status: (value: string, live: boolean) => void,
  ) {
    this.terminal.loadAddon(this.fit)
    this.terminal.open(element)
    this.resize()
    void document.fonts.ready.then(this.resize)
    const url = new URL(
      `/api/vms/${encodeURIComponent(id)}/console`,
      window.location.href,
    )
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    this.socket = new WebSocket(url)
    this.socket.binaryType = 'arraybuffer'
    this.socket.onopen = this.authenticate
    this.socket.onmessage = this.receive
    this.socket.onclose = this.closed
    this.socket.onerror = this.failed
    this.terminal.onData(this.send)
    this.observer = new ResizeObserver(this.resize)
    this.observer.observe(element)
  }

  private authenticate = () => {
    this.socket.send(JSON.stringify({ token: getToken() }))
    this.terminal.focus()
  }

  private receive = (event: MessageEvent) => {
    if (!this.active) return

    this.status('Connected', true)
    const replay = !this.replayed
    this.replayed = true
    this.replaying = this.replaying || replay
    const buffer = this.terminal.buffer.active
    const following = buffer.viewportY >= buffer.baseY
    this.terminal.write(
      event.data instanceof ArrayBuffer
        ? new Uint8Array(event.data)
        : event.data,
      () => {
        if (replay) this.replaying = false
        if (this.active && following) this.terminal.scrollToBottom()
      },
    )
  }

  private send = (data: string) => {
    if (!this.replaying && this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(new TextEncoder().encode(data))
    }
  }

  private resize = () => {
    if (!this.active) return

    cancelAnimationFrame(this.frame)
    this.frame = requestAnimationFrame(this.fitTerminal)
  }

  private fitTerminal = () => {
    if (!this.active) return

    const buffer = this.terminal.buffer.active
    const following = buffer.viewportY >= buffer.baseY
    this.fit.fit()
    if (following) this.terminal.scrollToBottom()
  }

  scrollToLatest() {
    this.terminal.scrollToBottom()
    this.terminal.focus()
  }
  private failed = () => {
    if (this.active) this.status('Connection failed', false)
  }
  private closed = (event: CloseEvent) => {
    if (this.active) this.status(event.reason || 'Disconnected', false)
  }

  dispose() {
    this.active = false
    cancelAnimationFrame(this.frame)
    this.observer.disconnect()
    this.socket.close()
    this.terminal.dispose()
  }
}

export function SerialConsole({ id, autoConnect = false }: SerialConsoleProps) {
  const releaseButton = useRef<HTMLButtonElement>(null)
  const container = useRef<HTMLDivElement>(null)
  const sessionRef = useRef<ConsoleSession | null>(null)
  const [connected, setConnected] = useState(autoConnect)
  const [status, setStatus] = useState('Disconnected')

  function connectConsole() {
    if (!connected || !container.current) return

    setStatus('Connecting…')
    const session = new ConsoleSession(container.current, id, updateStatus)
    sessionRef.current = session
    return () => {
      sessionRef.current = null
      session.dispose()
    }
  }

  useEffect(connectConsole, [id, connected])

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
        <p role="status" className="text-sm text-muted-foreground">
          {status}
        </p>
        <div className="flex flex-wrap gap-2">
          <Button ref={releaseButton} variant="outline" onClick={connected ? disconnect : connect}>
            {connected ? 'Disconnect' : 'Connect'}
          </Button>
        </div>
      </div>
      <div className="h-[min(28rem,50dvh)] min-w-0 overflow-hidden terminal-surface rounded-xl p-3">
        <div
          data-console-input
          ref={container}
          className="h-full w-full min-w-0"
          aria-label="VM serial terminal"
        />
      </div>
      <Button
        variant="ghost"
        disabled={!connected}
        onClick={() => sessionRef.current?.scrollToLatest()}
      >
        Scroll to End
      </Button>
    </section>
  )
}
