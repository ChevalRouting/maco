import { useState } from 'react'
import { EmptyState, PageHeader, Table, TableHeader, TableBody, TableRow, TableHead } from 'maco-ui'
import { CircleEllipsis, MoreVertical, Power, RotateCcw, Server } from 'lucide-react'
import { listVMs } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { CreateVMSheet } from '../components/CreateVMSheet'
import { HostPowerDialog, type HostPowerMode } from '../components/HostPowerDialog'
import { VMRow } from '../components/VMRow'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'
import { useSession } from '../hooks/useSession'
import { Menu } from '../components/Menu'
import { VMList } from '../components/VMList'
import { CreateButton } from '../components/CreateButton'
import { useCompact, useNarrow } from '../hooks/useNarrow'

export default function VMs() {
  const { admin } = useSession()
  const [creating, setCreating] = useState(false)
  const [powerMode, setPowerMode] = useState<HostPowerMode | null>(null)
  const narrow = useNarrow()
  const compact = useCompact()
  const vms = useResource(listVMs, [], 'vms')
  const action = useJobAction()
  const sorted = [...vms.data].sort((a, b) => a.manifest.name.localeCompare(b.manifest.name))

  return (
    <div className="space-y-6">
      <PageHeader
        title="Virtual Machines"
        action={
          admin ? (
            <div className="flex items-center gap-2">
              <Menu
                label="Host Power"
                className={compact ? 'h-9 w-9 text-primary hover:text-primary' : 'h-8 w-8'}
                sections={[
                  [
                    { key: 'reboot', label: 'Reboot Host…', icon: RotateCcw, onSelect: () => setPowerMode('reboot') },
                    { key: 'poweroff', label: 'Power Off Host…', icon: Power, destructive: true, onSelect: () => setPowerMode('poweroff') },
                  ],
                ]}
              >
                {compact ? <CircleEllipsis aria-hidden="true" className="h-6 w-6" /> : <MoreVertical aria-hidden="true" className="h-4 w-4" />}
              </Menu>
              <CreateButton toolbar label="Create Virtual Machine…" onClick={() => setCreating(true)} />
            </div>
          ) : undefined
        }
      />
      <ResourceNotice resource={vms} name="virtual machines" />
      <JobNotice error={action.error} />
      {!vms.loading && (vms.data.length ? (
        narrow ? <VMList vms={sorted} run={action.run} /> : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="hidden sm:table-cell"><span className="sr-only">Display</span></TableHead>
              <TableHead>Name</TableHead>
              <TableHead>State</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>vCPUs</TableHead>
              <TableHead>Memory</TableHead>
              <TableHead><span className="sr-only">Actions</span></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sorted.map((vm) => (
              <VMRow key={vm.manifest.id} vm={vm} run={action.run} />
            ))}
          </TableBody>
        </Table>
        )
      ) : (
        !vms.error && (
          <EmptyState icon={<Server />} title="No Virtual Machines" action={admin ? <CreateButton label="Create Virtual Machine…" onClick={() => setCreating(true)} /> : undefined} />
        )
      ))}
      {creating && <CreateVMSheet onClose={() => setCreating(false)} />}
      {powerMode && <HostPowerDialog mode={powerMode} onClose={() => setPowerMode(null)} />}
    </div>
  )
}
