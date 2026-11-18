import { useState } from 'react'
import { AlertDialog, Button, ContextMenu, cn, type ContextMenuPoint, type MenuItem } from 'maco-ui'
import { ChevronDown, Monitor, Play, Power, RotateCcw, RotateCw, Square, Terminal, Trash2 } from 'lucide-react'
import { deleteVM, rebootVM, shutdownVM, startVM, stopVM, type Job, type VMView } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { useSession } from '../hooks/useSession'
import { VMConsoleDialog, type ConsoleMode } from './VMConsoleDialog'
import { Menu } from './Menu'
import { StableWidth } from './StableWidth'

const powerLabels = ['Start', 'Starting…', 'Shut Down', 'Shutting Down…']

function powerLabel(label: string) {
  const Icon = label === 'Start' || label === 'Starting…' ? Play : Power
  return (
    <span className="inline-flex items-center gap-2">
      <Icon aria-hidden="true" className="h-4 w-4" />
      {label}
    </span>
  )
}

type Confirmation = 'stop' | 'reboot' | 'delete'

interface VMActionsProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
  emphasized?: boolean
  compact?: boolean
  contextPoint?: ContextMenuPoint | null
  onContextClose?: () => void
}

export function VMActions({ vm, run, emphasized, compact, contextPoint = null, onContextClose = () => {} }: VMActionsProps) {
  const { id, name } = vm.manifest
  const running = vm.phase === 'running'
  const { admin } = useSession()
  const action = useRowAction(id, name, 'vm', run)
  const [confirming, setConfirming] = useState<Confirmation | null>(null)
  const [consoleMode, setConsoleMode] = useState<ConsoleMode | null>(null)

  if (!admin) return null

  async function reboot() {
    await action.execute('vm.reboot', () => rebootVM(id))
  }

  async function power() {
    await action.execute(running ? 'vm.shutdown' : 'vm.start', () => (running ? shutdownVM(id) : startVM(id)))
  }

  async function confirm() {
    const job =
      confirming === 'stop'
        ? await action.execute('vm.stop', () => stopVM(id))
        : confirming === 'reboot'
          ? await action.execute('vm.reboot', () => rebootVM(id, true))
          : await action.execute('vm.delete', () => deleteVM(id))
    if (job) setConfirming(null)
  }

  const sections: MenuItem[][] = [
    [
      { key: 'reboot', label: 'Reboot', icon: RotateCw, disabled: !running || action.busy, onSelect: () => void reboot() },
      { key: 'reset', label: 'Force Reboot…', icon: RotateCcw, disabled: !running, onSelect: () => setConfirming('reboot') },
      { key: 'stop', label: 'Force Stop…', icon: Square, disabled: !running, onSelect: () => setConfirming('stop') },
    ],
    [
      { key: 'graphical', label: 'Graphical Console', icon: Monitor, disabled: !running, onSelect: () => setConsoleMode('graphical') },
      { key: 'serial', label: 'Serial Console', icon: Terminal, disabled: !running, onSelect: () => setConsoleMode('serial') },
    ],
    [{ key: 'delete', label: 'Delete…', icon: Trash2, destructive: true, disabled: running, onSelect: () => setConfirming('delete') }],
  ]

  const powerBusy = action.action === 'vm.start' || action.action === 'vm.shutdown'
  const powerText = powerBusy ? (running ? 'Shutting Down…' : 'Starting…') : running ? 'Shut Down' : 'Start'
  const suggested = emphasized && !running
  const powerItem: MenuItem = { key: 'power', label: running ? 'Shut Down' : 'Start', icon: running ? Power : Play, disabled: action.busy, onSelect: () => void power() }

  return (
    <div className="inline-flex items-center">
      {compact ? (
        <Button
          variant={suggested ? 'suggested' : 'default'}
          disabled={action.busy}
          onClick={power}
          aria-label={powerText}
          title={powerText}
          className="w-9 rounded-r-none px-0"
        >
          {running ? <Power aria-hidden="true" className="h-4 w-4" /> : <Play aria-hidden="true" className="h-4 w-4" />}
        </Button>
      ) : (
        <Button variant={suggested ? 'suggested' : 'default'} disabled={action.busy} onClick={power} className="rounded-r-none">
          <StableWidth value={powerText} values={powerLabels} render={powerLabel} />
        </Button>
      )}
      <Menu
        label={`More Actions for ${name}`}
        sections={sections}
        className={cn(
          'h-8 w-7 rounded-l-none border-l',
          suggested
            ? 'border-primary-foreground/20 bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground hover:brightness-105'
            : 'border-foreground/10 bg-secondary text-foreground [box-shadow:var(--control-shadow)] hover:bg-secondary hover:text-foreground hover:brightness-[0.97] dark:hover:brightness-110',
        )}
      >
        <ChevronDown aria-hidden="true" className="h-3.5 w-3.5" />
      </Menu>
      <ContextMenu label={`Actions for ${name}`} sections={[[powerItem], ...sections]} point={contextPoint} onClose={onContextClose} />

      <AlertDialog
        open={confirming === 'stop'}
        onCancel={() => setConfirming(null)}
        onConfirm={confirm}
        busy={action.busy}
        destructive
        title={`Force Stop ${name}?`}
        description="The guest is powered off without shutting down; unsaved data is lost"
        confirmLabel="Force Stop"
      />
      <AlertDialog
        open={confirming === 'reboot'}
        onCancel={() => setConfirming(null)}
        onConfirm={confirm}
        busy={action.busy}
        destructive
        title={`Force Reboot ${name}?`}
        description="The guest is reset without shutting down; unsaved data is lost"
        confirmLabel="Force Reboot"
      />
      <AlertDialog
        open={confirming === 'delete'}
        onCancel={() => setConfirming(null)}
        onConfirm={confirm}
        busy={action.busy}
        destructive
        title={`Delete ${name}?`}
        description="The virtual machine and all its disks are permanently deleted"
        confirmLabel="Delete Virtual Machine"
      />
      {running && consoleMode && (
        <VMConsoleDialog id={id} name={name} mode={consoleMode} onClose={() => setConsoleMode(null)} />
      )}
    </div>
  )
}
