import { useEffect, useId, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { AlertDialog, Button } from 'cheval-ui'
import { ChevronDown, LoaderCircle, Monitor, Play, Power, RotateCcw, Square, Terminal, Trash2 } from 'lucide-react'
import { deleteVM, rebootVM, shutdownVM, startVM, stopVM, type Job, type VMView } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { useSession } from '../hooks/useSession'
import { ActionStatus } from './ActionStatus'
import { VMConsoleDialog, type ConsoleMode } from './VMConsoleDialog'

export function VMActions({ vm, run }: { vm: VMView; run: (action: () => Promise<Job>) => Promise<Job | null> }) {
  const { id, name } = vm.manifest
  const running = vm.phase === 'running'
  const { admin } = useSession()
  const action = useRowAction(id, name, 'vm', run)
  const [open, setOpen] = useState(false)
  const [forceStop, setForceStop] = useState(false)
  const [reboot, setReboot] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const [consoleMode, setConsoleMode] = useState<ConsoleMode | null>(null)
  const [position, setPosition] = useState({ top: 0, left: 0 })
  const buttonGroup = useRef<HTMLDivElement>(null)
  const trigger = useRef<HTMLButtonElement>(null)
  const menu = useRef<HTMLDivElement>(null)
  const menuId = useId()
  const Icon = action.busy ? LoaderCircle : running ? Power : Play

  function close() {
    setOpen(false)
    trigger.current?.focus()
  }

  function toggleMenu() {
    if (open) return close()
    const rect = buttonGroup.current?.getBoundingClientRect()
    if (rect) {
      const menuWidth = 208
      const margin = 8
      const fitsRight = rect.left + menuWidth <= window.innerWidth - margin
      const left = fitsRight ? rect.left : rect.right - menuWidth
      const estimatedHeight = running ? 230 : 150
      const top = rect.bottom + estimatedHeight + margin < window.innerHeight ? rect.bottom + 6 : Math.max(margin, rect.top - estimatedHeight - 6)
      setPosition({ top, left: Math.max(margin, Math.min(left, window.innerWidth - menuWidth - margin)) })
    }
    setOpen(true)
  }

  useEffect(() => {
    if (!open) return
    menu.current?.querySelector<HTMLButtonElement>('button:not(:disabled)')?.focus()
    function dismiss(event: MouseEvent) {
      if (!menu.current?.contains(event.target as Node) && !trigger.current?.contains(event.target as Node)) setOpen(false)
    }
    function hide() { setOpen(false) }
    document.addEventListener('mousedown', dismiss)
    window.addEventListener('resize', hide)
    window.addEventListener('scroll', hide, true)
    return () => {
      document.removeEventListener('mousedown', dismiss)
      window.removeEventListener('resize', hide)
      window.removeEventListener('scroll', hide, true)
    }
  }, [open])

  async function power() {
    await action.execute(running ? 'vm.shutdown' : 'vm.start', () => running ? shutdownVM(id) : startVM(id))
  }

  async function remove() {
    const job = await action.execute('vm.delete', () => deleteVM(id))
    if (job) setConfirm(false)
  }

  const itemClass = 'flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-sm hover:bg-muted focus:bg-muted focus:outline-none disabled:pointer-events-none disabled:opacity-50'

  if (!admin) {
    return <span className="text-sm text-muted-foreground">Read only</span>
  }

  return (
    <>
      <div className="inline-flex flex-wrap items-center gap-2">
        <div ref={buttonGroup} role="group" aria-label={`Actions for ${name}`} className="inline-flex">
          <Button
            size="sm"
            variant={running ? 'outline' : 'suggested'}
            className="rounded-r-none"
            disabled={action.busy}
            onClick={power}
            aria-busy={action.busy || undefined}
          >
            <Icon aria-hidden="true" className={`mr-2 h-4 w-4 ${action.busy ? 'animate-spin' : ''}`} />
            {action.busy ? 'Working…' : running ? 'Shut Down' : 'Start'}
          </Button>
          <Button
            ref={trigger}
            size="sm"
            variant={running ? 'outline' : 'suggested'}
            className="rounded-l-none border-l border-l-current/20 px-2"
            disabled={action.busy}
            onClick={toggleMenu}
            aria-label={`More actions for ${name}`}
            aria-haspopup="menu"
            aria-expanded={open}
            aria-controls={open ? menuId : undefined}
            onKeyDown={(event) => {
              if (event.key === 'ArrowDown') {
                event.preventDefault()
                if (!open) toggleMenu()
              }
            }}
          >
            <ChevronDown aria-hidden="true" className="h-4 w-4" />
          </Button>
        </div>
        <ActionStatus job={action.job} error={action.error} hideProgress />
      </div>

      {open && createPortal(
        <div
          ref={menu}
          id={menuId}
          role="menu"
          aria-label={`Actions for ${name}`}
          style={position}
          className="fixed z-50 max-h-[calc(100dvh-1rem)] overflow-y-auto w-52 rounded-lg border bg-card p-1 shadow-lg"
          onKeyDown={(event) => {
            if (event.key === 'Escape') {
              event.preventDefault()
              close()
            }
            if (event.key === 'Tab') {
              setOpen(false)
            }
            if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
              event.preventDefault()
              const items = Array.from(
                menu.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') || [],
              )
              const current = items.indexOf(document.activeElement as HTMLButtonElement)
              const next =
                event.key === 'Home'
                  ? 0
                  : event.key === 'End'
                    ? items.length - 1
                    : (current + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length
              items[next]?.focus()
            }
          }}
        >
          {running && (
            <button
              role="menuitem"
              className={itemClass}
              disabled={action.busy}
              onClick={() => {
                close()
                setReboot(true)
              }}
            >
              <RotateCcw className="h-4 w-4" />
              Reboot…
            </button>
          )}
          {running && (
            <button
              role="menuitem"
              className={itemClass}
              disabled={action.busy}
              onClick={() => {
                close()
                setForceStop(true)
              }}
            >
              <Square className="h-4 w-4" />
              Force Stop…
            </button>
          )}
          <button
            role="menuitem"
            className={itemClass}
            disabled={!running || action.busy}
            onClick={() => {
              close()
              setConsoleMode('graphical')
            }}
          >
            <Monitor className="h-4 w-4" />
            Graphical console
          </button>
          <button
            role="menuitem"
            className={itemClass}
            disabled={!running || action.busy}
            onClick={() => {
              close()
              setConsoleMode('serial')
            }}
          >
            <Terminal className="h-4 w-4" />
            Serial console
          </button>
          <div role="separator" className="my-1 border-t" />
          <button
            role="menuitem"
            className={`${itemClass} text-destructive`}
            disabled={running || action.busy}
            onClick={() => {
              close()
              setConfirm(true)
            }}
          >
            <Trash2 className="h-4 w-4" />
            Delete…
          </button>
        </div>,
        document.body,
      )}

      <AlertDialog
        open={forceStop}
        onCancel={() => setForceStop(false)}
        onConfirm={async () => {
          const job = await action.execute('vm.stop', () => stopVM(id))
          if (job) setForceStop(false)
        }}
        busy={action.busy}
        destructive
        title={`Force Stop ${name}?`}
        description="This immediately stops the VM. Unsaved work inside the guest may be lost. Try Shut Down first."
        confirmLabel="Force Stop"
      />
      <AlertDialog
        open={reboot}
        onCancel={() => setReboot(false)}
        onConfirm={async () => {
          const job = await action.execute('vm.reboot', () => rebootVM(id))
          if (job) setReboot(false)
        }}
        busy={action.busy}
        title={`Reboot ${name}?`}
        description="This restarts the VM. Unsaved work inside the guest may be lost."
        confirmLabel="Reboot"
      />
      <AlertDialog
        open={confirm}
        onCancel={() => setConfirm(false)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title={`Delete ${name}?`}
        description="This permanently deletes the VM and all its virtual disks. Data inside those disks cannot be recovered through maco."
        confirmLabel="Delete"
      />
      {running && consoleMode && (
        <VMConsoleDialog id={id} name={name} mode={consoleMode} onClose={() => setConsoleMode(null)} />
      )}
    </>
  )
}
