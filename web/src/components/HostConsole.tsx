import { useEffect, useRef, useState } from 'react'
import { Button } from 'maco-ui'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { getToken } from '../api'
import { terminalTheme } from '../lib/terminalTheme'
import '@xterm/xterm/css/xterm.css'

class ShellSession {
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
  private ready = false
  private frame = 0

  constructor(
    element: HTMLElement,
    private status: (value: string, live: boolean) => void,
  ) {
    this.terminal.loadAddon(this.fit)
    this.terminal.open(element)
    this.resize()
    void document.fonts.ready.then(this.resize)
    const url = new URL('/api/host/console', window.location.href)
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
    this.ready = true
    this.sendSize()
    this.terminal.focus()
  }

  private receive = (event: MessageEvent) => {
    if (!this.active) return

    this.status('Connected', true)
    const buffer = this.terminal.buffer.active
    const following = buffer.viewportY >= buffer.baseY
    this.terminal.write(
      event.data instanceof ArrayBuffer
        ? new Uint8Array(event.data)
        : event.data,
      () => {
        if (this.active && following) this.terminal.scrollToBottom()
      },
    )
  }

  private send = (data: string) => {
    if (this.socket.readyState === WebSocket.OPEN) {
      this.socket.send(new TextEncoder().encode(data))
    }
  }

  private sendSize = () => {
    if (!this.ready || this.socket.readyState !== WebSocket.OPEN) return

    this.socket.send(
      JSON.stringify({ cols: this.terminal.cols, rows: this.terminal.rows }),
    )
  }

  private resize = () => {
    if (!this.active) return

    cancelAnimationFrame(this.frame)
    this.frame = requestAnimationFrame(this.fitTerminal)
  }

  private fitTerminal = () => {
    if (!this.active) return

    this.fit.fit()
    this.sendSize()
    this.terminal.scrollToBottom()
  }

  private failed = () => {
    if (this.active) this.status('Shell connection failed', false)
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

export function HostConsole() {
  const container = useRef<HTMLDivElement>(null)
  const sessionRef = useRef<ShellSession | null>(null)
  const [connected, setConnected] = useState(true)
  const [status, setStatus] = useState('Connecting…')

  function connectShell() {
    if (!connected || !container.current) return

    setStatus('Connecting…')
    const session = new ShellSession(container.current, updateStatus)
    sessionRef.current = session
    return () => {
      sessionRef.current = null
      session.dispose()
    }
  }

  useEffect(connectShell, [connected])

  function updateStatus(value: string, live: boolean) {
    setStatus(value)
    if (!live) setConnected(false)
  }

  return (
    <section className="min-w-0 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p role="status" className="text-sm text-muted-foreground">
          {status}
        </p>
        <Button variant="outline" onClick={() => setConnected(!connected)}>
          {connected ? 'Disconnect' : 'Reconnect'}
        </Button>
      </div>
      <div className="h-[min(38rem,70dvh)] min-w-0 overflow-hidden terminal-surface rounded-xl p-3">
        <div
          data-console-input
          ref={container}
          className="h-full w-full min-w-0"
          aria-label="Host shell terminal"
        />
      </div>
    </section>
  )
}
