import { useState, type ReactNode } from 'react'
import { CopyButton } from 'cheval-ui'
import { SourceName } from './SourceName'
import { memoryLabel } from '../ux'
import { VMStatusBadge } from './StatusBadge'
import { type Job, type VMView } from '../api'
import { VMPreview } from './VMPreview'
import { VMActions } from './VMActions'
import { VMConsoleDialog, type ConsoleMode } from './VMConsoleDialog'
import { GuestAgent } from './GuestAgent'
import { useSession } from '../hooks/useSession'

interface VMWorkspaceProps {
  vm: VMView
  networkName: string
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

interface CharacteristicProps {
  label: string
  value: ReactNode
}

function Characteristic({ label, value }: CharacteristicProps) {
  return (
    <div className="grid grid-cols-[7rem_minmax(0,1fr)] gap-3 py-2 text-sm">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="break-words font-medium">{value}</dd>
    </div>
  )
}

export function VMWorkspace({ vm, run, networkName }: VMWorkspaceProps) {
  const manifest = vm.manifest
  const running = vm.phase === 'running'
  const { admin } = useSession()
  const [consoleMode, setConsoleMode] = useState<ConsoleMode | null>(null)

  return (
    <div className="space-y-6">
      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
        <section
          className="min-w-0 space-y-4"
          aria-label="VM preview and actions"
        >
          <div
            role="group"
            aria-label="Quick actions"
            className="flex flex-wrap items-center gap-2"
          >
            <VMActions vm={vm} run={run} />
            <div className="ml-auto">
              <VMStatusBadge phase={vm.phase} />
            </div>
          </div>
          <VMPreview
            id={manifest.id}
            name={manifest.name}
            running={running}
            bootTime={vm.boot_time}
            large
            onOpenConsole={admin ? setConsoleMode : undefined}
          />
          <p className="text-xs text-muted-foreground">
            {running
              ? admin
                ? 'Latest screenshot. Hover to open the graphical or serial console.'
                : 'Latest screenshot, updated when a new capture is available.'
              : 'Start the VM to see its display preview.'}
          </p>
        </section>
        <aside
          className="rounded-xl border bg-card p-4"
          aria-label="VM characteristics"
        >
          <h2 className="mb-2 font-semibold">Virtual machine</h2>
          <dl className="divide-y divide-border">
            {manifest.image && <Characteristic label="Source" value={<SourceName image={manifest.image} />} />}
            <Characteristic label="CPUs" value={manifest.cpus} />
            <Characteristic
              label="Memory"
              value={memoryLabel(manifest.memory_mib)}
            />
            <Characteristic
              label="Disk capacity"
              value={`${manifest.disk_size_gib + (manifest.disks || []).reduce((total, disk) => total + disk.size_gib, 0)} GiB`}
            />
            <Characteristic
              label="Disks"
              value={(manifest.disks?.length || 0) + 1}
            />
            <Characteristic
              label="Network"
              value={networkName}
            />
            <Characteristic
              label="Automatic Startup"
              value={manifest.autostart ? 'Enabled' : 'Disabled'}
            />
            {manifest.tags && manifest.tags.length > 0 && (
              <Characteristic
                label="Tags"
                value={
                  <span className="flex flex-wrap gap-1.5">
                    {manifest.tags.map((tag) => (
                      <span key={tag} className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        {tag}
                      </span>
                    ))}
                  </span>
                }
              />
            )}
          </dl>
          <details className="mt-4 text-sm">
            <summary className="cursor-pointer text-muted-foreground">Technical Details</summary>
            <dl className="mt-2">
              <Characteristic
                label="VM ID"
                value={
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="break-all font-mono">{manifest.id}</span>
                    <CopyButton text={manifest.id} label="Copy VM ID" />
                  </span>
                }
              />
              <Characteristic label="Process ID" value={vm.pid || 'Not Running'} />
            </dl>
          </details>
        </aside>
      </div>

      {running && <GuestAgent id={manifest.id} />}

      {running && consoleMode && (
        <VMConsoleDialog
          id={manifest.id}
          name={manifest.name}
          mode={consoleMode}
          onClose={() => setConsoleMode(null)}
        />
      )}
    </div>
  )
}
