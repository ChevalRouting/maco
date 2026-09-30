import { useState } from 'react'
import {
  EmptyState,
  PageHeader,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from 'cheval-ui'
import { Server, Power, RotateCcw } from 'lucide-react'
import { listVMs, getHost, getDiskStorage } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { CreateVMChooser } from '../components/CreateVMChooser'
import { HostPowerDialog, type HostPowerMode } from '../components/HostPowerDialog'
import { HostStatCards } from '../components/HostStatCards'
import { SplitButton } from '../components/SplitButton'
import { Button } from 'cheval-ui'
import { VMRow } from '../components/VMRow'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'
import { useSession } from '../hooks/useSession'

export default function VMs() {
  const { admin } = useSession()
  const [creating, setCreating] = useState(false)
  const [powerMode, setPowerMode] = useState<HostPowerMode | null>(null)
  const vms = useResource(listVMs, [], 'vms')
  const host = useResource(getHost, null, 'host')
  const storage = useResource(getDiskStorage, null, 'storage')
  const action = useJobAction()

  return (
    <div className="space-y-6">
      <PageHeader
        title="Virtual machines"
        action={
          admin ? (
            <div className="flex flex-wrap items-center gap-2">
              <SplitButton
                label="Create virtual machine"
                onClick={() => setCreating(true)}
              />
              <Button variant="outline" onClick={() => setPowerMode('reboot')}>
                <RotateCcw className="h-4 w-4 shrink-0" />
                Reboot host
              </Button>
              <Button variant="destructive" onClick={() => setPowerMode('poweroff')}>
                <Power className="h-4 w-4 shrink-0" />
                Power off host
              </Button>
            </div>
          ) : undefined
        }
      />
      <ResourceNotice resource={host} name="host capacity" />
      <ResourceNotice resource={storage} name="host storage" />
      {host.data && (
        <HostStatCards host={host.data} vms={vms.data} storage={storage.data} />
      )}
      <ResourceNotice resource={vms} name="virtual machines" />
      <JobNotice error={action.error} />
      {!vms.loading && (vms.data.length ? (
        <Table className="table-cards">
          <TableHeader>
            <TableRow>
              <TableHead>Preview</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>Memory</TableHead>
              <TableHead>State</TableHead>
              <TableHead>Automatic Startup</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {vms.data.map((vm) => (
              <VMRow key={vm.manifest.id} vm={vm} run={action.run} />
            ))}
          </TableBody>
        </Table>
      ) : (
        !vms.error && (
          <EmptyState
            icon={<Server />}
            title="No virtual machines yet"
            message={
              admin
                ? 'Create your first virtual machine to get started.'
                : 'An administrator can create virtual machines.'
            }
          />
        )
      ))}
      {creating && <CreateVMChooser onClose={() => setCreating(false)} />}
      {powerMode && (
        <HostPowerDialog mode={powerMode} onClose={() => setPowerMode(null)} />
      )}
    </div>
  )
}
