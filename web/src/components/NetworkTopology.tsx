import { type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { PreferencesGroup, Row, cn } from 'maco-ui'
import { EthernetPort, Monitor } from 'lucide-react'
import { type Topology, type TopologyPort, type TopologyRow, type TopologyVM } from '../lib/topology'

interface NetworkTopologyProps {
  topology: Topology
  vmsLoading?: boolean
}

interface DotProps {
  on?: boolean
}

interface PortChipProps {
  port: TopologyPort
}

interface VMChipProps {
  vm: TopologyVM
}

interface EndpointProps {
  side: 'left' | 'right'
  children: ReactNode
}

interface SwitchProps {
  row: TopologyRow
  vmsLoading?: boolean
}

interface EmptyProps {
  side: 'left' | 'right'
  children: ReactNode
}

const chip = 'inline-flex min-w-0 max-w-full items-center gap-1.5 rounded-md border border-border bg-background px-2 py-1 text-xs'

function Dot({ on }: DotProps) {
  return <span aria-hidden="true" className={cn('h-2 w-2 shrink-0 rounded-full', on ? 'bg-success' : 'bg-muted-foreground/40')} />
}

function PortChip({ port }: PortChipProps) {
  return (
    <span className={chip} title={[port.hardware, port.up === undefined ? 'Interface not found' : port.up ? 'Link up' : 'Link down'].filter(Boolean).join(' · ')}>
      <EthernetPort aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <span className="truncate font-mono">{port.device}</span>
      {(port.detail || port.hardware) && <span className="truncate text-muted-foreground">{port.detail || port.hardware}</span>}
      <Dot on={port.up} />
    </span>
  )
}

function VMChip({ vm }: VMChipProps) {
  return (
    <Link to={`/vms/${vm.id}`} className={cn(chip, 'text-foreground transition-colors hover:bg-accent/50')} title={vm.running ? 'Running' : 'Stopped'}>
      <Monitor aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      <span className="truncate">{vm.name}</span>
      <Dot on={vm.running} />
    </Link>
  )
}

function Wire() {
  return <span aria-hidden="true" className="h-px w-5 shrink-0 bg-border" />
}

function Endpoint({ side, children }: EndpointProps) {
  return (
    <div className={cn('flex min-w-0 items-center', side === 'left' && 'justify-end')}>
      {side === 'right' && <Wire />}
      {children}
      {side === 'left' && <Wire />}
    </div>
  )
}

function Bus() {
  return <span aria-hidden="true" className="w-1 shrink-0 self-stretch rounded-full bg-muted-foreground/50" />
}

function Empty({ side, children }: EmptyProps) {
  return <p className={cn('px-5 py-1 text-xs text-muted-foreground', side === 'left' && 'text-right')}>{children}</p>
}

function uplinks(row: TopologyRow, side: 'left' | 'right'): ReactNode {
  if (row.ports.length) {
    return row.ports.map((port) => (
      <Endpoint key={`${port.device}-${port.detail ?? ''}`} side={side}>
        <PortChip port={port} />
      </Endpoint>
    ))
  }
  if (row.key === 'user') {
    return (
      <Endpoint side={side}>
        <span className={cn(chip, 'text-muted-foreground')}>Host NAT</span>
      </Endpoint>
    )
  }
  return <Empty side={side}>{row.network?.mode === 'switch' ? 'No uplink' : 'No adapter'}</Empty>
}

function guests(row: TopologyRow, side: 'left' | 'right', loading = false): ReactNode {
  if (!row.vms.length) return <Empty side={side}>{loading ? 'Loading VMs…' : 'No VMs'}</Empty>
  return row.vms.map((vm) => (
    <Endpoint key={vm.id} side={side}>
      <VMChip vm={vm} />
    </Endpoint>
  ))
}

function Switch({ row, vmsLoading }: SwitchProps) {
  return (
    <div className="px-4 py-3">
      <p className="truncate text-sm">
        {row.name}
        <span className="ml-2 text-xs text-muted-foreground">{[row.mode, row.device].filter(Boolean).join(' · ')}</span>
      </p>

      <SwitchDiagram row={row} vmsLoading={vmsLoading} />
    </div>
  )
}

export function SwitchDiagram({ row, vmsLoading }: SwitchProps) {
  return (
    <>
      <div className="mt-2 hidden grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:grid">
        <div className="flex min-w-0 flex-col justify-center gap-1.5 py-1">{guests(row, 'left', vmsLoading)}</div>
        <Bus />
        <div className="flex min-w-0 flex-col justify-center gap-1.5 py-1">{uplinks(row, 'right')}</div>
      </div>

      <div className="mt-2 flex md:hidden">
        <Bus />
        <div className="flex min-w-0 flex-1 flex-col gap-1.5 py-1">
          {guests(row, 'right', vmsLoading)}
          <span className="h-1" />
          {uplinks(row, 'right')}
        </div>
      </div>
    </>
  )
}

export function NetworkTopology({ topology, vmsLoading }: NetworkTopologyProps) {
  if (!topology.rows.length) return null

  return (
    <PreferencesGroup title="Topology">
      {topology.rows.map((row) => <Switch key={row.key} row={row} vmsLoading={vmsLoading} />)}
      {topology.unused.length > 0 && (
        <Row prefix={<EthernetPort className="h-4 w-4" />} title="Unused Interfaces">
          <div className="flex flex-wrap gap-1.5">
            {topology.unused.map((port) => <PortChip key={port.device} port={{ device: port.device, up: port.up, hardware: port.hardware_port }} />)}
          </div>
        </Row>
      )}
    </PreferencesGroup>
  )
}
