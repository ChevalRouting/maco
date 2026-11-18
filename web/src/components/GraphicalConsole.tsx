import { useEffect, useRef, useState } from 'react'
import { Button, Segmented } from 'maco-ui'
import { Hand, Keyboard, Maximize } from 'lucide-react'
import RFB from '@novnc/novnc'
import { getToken } from '../api'
import { ConsoleKeyboard, type ConsoleKeyboardHandle, type KeySink } from './ConsoleKeyboard'

interface GraphicalConsoleProps {
  id: string
  autoConnect?: boolean
}

type Zoom = 'fit' | 'actual'

interface View {
  zoom: Zoom
  pan: boolean
}

const zoomOptions = [
  { value: 'fit' as const, label: 'Fit' },
  { value: 'actual' as const, label: '100%' },
]

function touchScreen(): boolean {
  return typeof window !== 'undefined' && window.matchMedia('(pointer: coarse)').matches
}

function fullscreenAvailable(): boolean {
  return typeof document !== 'undefined' && document.fullscreenEnabled
}

class DisplaySession implements KeySink {
  private socket: WebSocket
  private display?: RFB
  private active = true
  private view: View = { zoom: 'fit', pan: false }

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
    this.display.resizeSession = false
    this.apply()
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
    if (this.active) this.status('Disconnected', false)
  }

  private closed = (event: CloseEvent) => {
    if (this.active) this.status(event.reason || 'Disconnected', false)
  }

  private failed = () => {
    if (this.active) this.status('Connection failed', false)
  }

  setView(view: View) {
    this.view = view
    this.apply()
  }

  private apply() {
    if (!this.display) return
    const actual = this.view.zoom === 'actual'
    this.display.scaleViewport = !actual
    this.display.clipViewport = actual
    this.display.dragViewport = actual && this.view.pan
  }

  sendKey(keysym: number, down?: boolean) {
    this.display?.sendKey(keysym, null, down)
  }

  sendCtrlAltDel() {
    this.display?.sendCtrlAltDel()
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
  const frame = useRef<HTMLDivElement>(null)
  const session = useRef<DisplaySession | null>(null)
  const keyboard = useRef<ConsoleKeyboardHandle>(null)
  const [sink, setSink] = useState<KeySink | null>(null)
  const [typing, setTyping] = useState(false)
  const [view, setView] = useState<View>({ zoom: 'fit', pan: false })
  const [connected, setConnected] = useState(autoConnect)
  const [status, setStatus] = useState('Disconnected')

  function connectDisplay() {
    if (!connected || !container.current) return

    setStatus('Connecting…')
    const current = new DisplaySession(container.current, id, updateStatus)
    current.setView(view)
    session.current = current
    setSink(current)
    return () => {
      current.dispose()
      session.current = null
      setSink(null)
      setTyping(false)
    }
  }

  useEffect(connectDisplay, [id, connected])

  function changeView(next: View) {
    setView(next)
    session.current?.setView(next)
  }

  function fullscreen() {
    if (document.fullscreenElement) void document.exitFullscreen()
    else void frame.current?.requestFullscreen()
  }

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
        <div className="flex flex-wrap items-center gap-2">
          <Segmented value={view.zoom} onChange={(zoom) => changeView({ ...view, zoom })} options={zoomOptions} />
          {view.zoom === 'actual' && (
            <Button
              size="icon"
              aria-label="Drag to Pan"
              title="Drag to Pan"
              aria-pressed={view.pan}
              variant={view.pan ? 'suggested' : 'outline'}
              onClick={() => changeView({ ...view, pan: !view.pan })}
            >
              <Hand aria-hidden="true" className="h-4 w-4" />
            </Button>
          )}
          {connected && touchScreen() && (
            <Button
              variant={typing ? 'suggested' : 'outline'}
              size="icon"
              aria-label="Keyboard"
              title="Keyboard"
              aria-pressed={typing}
              onClick={() => {
                setTyping(!typing)
                if (!typing) keyboard.current?.open()
              }}
            >
              <Keyboard aria-hidden="true" className="h-4 w-4" />
            </Button>
          )}
          {fullscreenAvailable() && (
            <Button variant="outline" size="icon" aria-label="Full Screen" title="Full Screen" onClick={fullscreen}>
              <Maximize aria-hidden="true" className="h-4 w-4" />
            </Button>
          )}
          <Button ref={releaseButton} variant="outline" onClick={connected ? disconnect : connect}>
            {connected ? 'Disconnect' : 'Connect'}
          </Button>
        </div>
      </div>
      <ConsoleKeyboard ref={keyboard} sink={sink} visible={typing && connected} />
      <div ref={frame} className="rounded-xl border border-border bg-background p-1 [&:fullscreen]:rounded-none [&:fullscreen]:border-0 [&:fullscreen]:p-0">
        <div
          data-console-input
          ref={container}
          className="h-[min(32rem,55dvh)] min-h-0 w-full min-w-0 overflow-hidden [:fullscreen>&]:h-full"
          aria-label="VM graphical display"
        />
      </div>
    </section>
  )
}
