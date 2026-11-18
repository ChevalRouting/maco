import { useState } from 'react'
import { Button, CopyButton, PreferencesGroup, Row } from 'maco-ui'
import { Monitor, Terminal } from 'lucide-react'
import { type VMView } from '../api'
import { memoryLabel } from '../ux'
import { useSession } from '../hooks/useSession'
import { GuestAgent } from './GuestAgent'
import { SourceName } from './SourceName'
import { VMStateChip } from './StateChips'
import { VMConsoleDialog, type ConsoleMode } from './VMConsoleDialog'
import { VMPreview } from './VMPreview'

interface VMOverviewProps {
  vm: VMView
  networkName: string
}

export function VMOverview({ vm, networkName }: VMOverviewProps) {
  const manifest = vm.manifest
  const running = vm.phase === 'running'
  const { admin } = useSession()
  const [consoleMode, setConsoleMode] = useState<ConsoleMode | null>(null)
  const capacity = manifest.disk_size_gib + (manifest.disks || []).reduce((total, disk) => total + disk.size_gib, 0)

  return (
    <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]">
      <section className="min-w-0 space-y-3" aria-label="Display">
        <VMPreview id={manifest.id} name={manifest.name} running={running} bootTime={vm.boot_time} large />
        {admin && (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Button className="gap-2" disabled={!running} onClick={() => setConsoleMode('graphical')}>
              <Monitor aria-hidden="true" className="h-4 w-4" />
              Graphical Console
            </Button>
            <Button className="gap-2" disabled={!running} onClick={() => setConsoleMode('serial')}>
              <Terminal aria-hidden="true" className="h-4 w-4" />
              Serial Console
            </Button>
          </div>
        )}
      </section>

      <div className="min-w-0 space-y-6">
        <PreferencesGroup title="Details">
          <Row title="State">
            <VMStateChip phase={vm.phase} align="end" />
          </Row>
          <Row title="Source">
            <span className="text-sm">
              <SourceName image={manifest.image} />
            </span>
          </Row>
          <Row title="vCPUs">
            <span className="text-sm tabular-nums">{manifest.cpus}</span>
          </Row>
          <Row title="Memory">
            <span className="text-sm tabular-nums">{memoryLabel(manifest.memory_mib)}</span>
          </Row>
          <Row title="Disks">
            <span className="text-sm tabular-nums">{`${(manifest.disks?.length || 0) + 1} · ${capacity} GiB`}</span>
          </Row>
          <Row title="Network">
            <span className="text-sm">{networkName}</span>
          </Row>
          <Row title="Start Automatically">
            <span className="text-sm">{manifest.autostart ? 'On' : 'Off'}</span>
          </Row>
          {manifest.tags && manifest.tags.length > 0 && (
            <Row title="Tags">
              <span className="text-sm">{manifest.tags.join(', ')}</span>
            </Row>
          )}
          <Row title="ID">
            <span className="break-all font-mono text-xs">{manifest.id}</span>
            <CopyButton text={manifest.id} label="Copy ID" />
          </Row>
          {running && (
            <Row title="Process ID">
              <span className="font-mono text-sm">{vm.pid}</span>
            </Row>
          )}
        </PreferencesGroup>
        {running && <GuestAgent id={manifest.id} />}
      </div>

      {running && consoleMode && (
        <VMConsoleDialog id={manifest.id} name={manifest.name} mode={consoleMode} onClose={() => setConsoleMode(null)} />
      )}
    </div>
  )
}
