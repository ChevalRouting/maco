import { useEffect, useRef, useState } from 'react'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

interface TerminalPaneProps {
  wsPath: string
  token?: string
  className?: string
  onOpen?: () => void
  onClose?: (clean: boolean) => void
  quiet?: boolean
  keys?: boolean
}

interface TerminalKey {
  label: string
  title: string
  data: string
}

const terminalKeys: TerminalKey[] = [
  { label: 'Esc', title: 'Escape', data: '\x1b' },
  { label: 'Tab', title: 'Tab', data: '\t' },
  { label: '^C', title: 'Ctrl+C', data: '\x03' },
  { label: '^D', title: 'Ctrl+D', data: '\x04' },
  { label: '^Z', title: 'Ctrl+Z', data: '\x1a' },
  { label: '^L', title: 'Ctrl+L', data: '\x0c' },
  { label: '^R', title: 'Ctrl+R', data: '\x12' },
  { label: '↑', title: 'Up', data: '\x1b[A' },
  { label: '↓', title: 'Down', data: '\x1b[B' },
  { label: '←', title: 'Left', data: '\x1b[D' },
  { label: '→', title: 'Right', data: '\x1b[C' },
  { label: '|', title: 'Pipe', data: '|' },
  { label: '~', title: 'Tilde', data: '~' },
  { label: '/', title: 'Slash', data: '/' },
  { label: '-', title: 'Dash', data: '-' },
]

function withCtrl(data: string) {
  if (data.length !== 1) {
    return data
  }
  const code = data.toUpperCase().charCodeAt(0)
  if (code >= 64 && code <= 95) {
    return String.fromCharCode(code - 64)
  }
  return data
}

const keyClass =
  'shrink-0 rounded-md bg-white/10 px-2.5 py-1.5 font-mono text-xs text-zinc-200 transition-colors hover:bg-white/20 active:bg-white/30'

export function TerminalPane({
  wsPath,
  token,
  className,
  onOpen,
  onClose,
  quiet,
  keys = false,
}: TerminalPaneProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)
  const sendRef = useRef<(data: string) => void>(() => {})
  const ctrlRef = useRef(false)
  const [ctrl, setCtrl] = useState(false)
  const onOpenRef = useRef(onOpen)
  const onCloseRef = useRef(onClose)
  const quietRef = useRef(quiet)
  const tokenRef = useRef(token)
  onOpenRef.current = onOpen
  onCloseRef.current = onClose
  quietRef.current = quiet
  tokenRef.current = token

  useEffect(() => {
    const el = containerRef.current
    if (!el) {
      return
    }

    let unmounting = false

    const term = new Terminal({
      theme: {
        background: '#09090b',
        foreground: '#e4e4e7',
        cursor: '#a1a1aa',
        selectionBackground: '#3f3f46',
      },
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
      fontSize: 13,
      lineHeight: 1.4,
      cursorBlink: true,
      convertEol: true,
      scrollback: 5000,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(el)
    fitAddon.fit()
    termRef.current = term

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = tokenRef.current ?? ''
    const sep = wsPath.includes('?') ? '&' : '?'
    const wsUrl = `${proto}//${window.location.host}${wsPath}${sep}token=${encodeURIComponent(token)}`

    let ws: WebSocket | null = null

    const sendResize = () => {
      fitAddon.fit()
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }

    const openTimer = setTimeout(() => {
      if (unmounting) {
        return
      }
      ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'

      ws.onmessage = (e) => {
        if (e.data instanceof ArrayBuffer) {
          term.write(new Uint8Array(e.data))
        } else {
          term.write(e.data as string)
        }
      }

      ws.onclose = (e) => {
        if (!quietRef.current) {
          term.write('\r\n\x1b[2m[connection closed]\x1b[0m\r\n')
        }
        if (!unmounting) {
          onCloseRef.current?.(e.code === 1000 || e.code === 1005)
        }
      }

      ws.onerror = () => {
        if (!quietRef.current) {
          term.write('\r\n\x1b[31m[connection error]\x1b[0m\r\n')
        }
      }

      ws.onopen = () => {
        sendResize()
        onOpenRef.current?.()
      }
    }, 0)

    const send = (data: string) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }))
      }
    }
    sendRef.current = send

    term.onData((data) => {
      if (ctrlRef.current) {
        ctrlRef.current = false
        setCtrl(false)
        send(withCtrl(data))
        return
      }
      send(data)
    })

    const ro = new ResizeObserver(sendResize)
    ro.observe(el)

    return () => {
      unmounting = true
      clearTimeout(openTimer)
      ro.disconnect()
      ws?.close()
      termRef.current = null
      sendRef.current = () => {}
      term.dispose()
    }
  }, [wsPath])

  return (
    <div
      className={className}
      style={{
        height: '100%',
        background: '#09090b',
        padding: '10px 12px',
        boxSizing: 'border-box',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div
        ref={containerRef}
        style={{ flex: 1, minHeight: 0 }}
      />
      {keys && (
        <div
          className="-mx-3 -mb-2.5 mt-2 flex gap-1.5 overflow-x-auto overscroll-x-contain border-t border-white/10 px-3 py-2"
        >
          <button
            type="button"
            aria-label="Ctrl"
            aria-pressed={ctrl}
            title="Ctrl, applied to the next key"
            className={`${keyClass} ${ctrl ? 'bg-zinc-200 text-zinc-900 hover:bg-zinc-200' : ''}`}
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => {
              ctrlRef.current = !ctrlRef.current
              setCtrl(ctrlRef.current)
              termRef.current?.focus()
            }}
          >
            Ctrl
          </button>
          {terminalKeys.map((key) => (
            <button
              key={key.title}
              type="button"
              aria-label={key.title}
              title={key.title}
              className={keyClass}
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => {
                const data = ctrlRef.current ? withCtrl(key.data) : key.data
                ctrlRef.current = false
                setCtrl(false)
                sendRef.current(data)
                termRef.current?.focus()
              }}
            >
              {key.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
