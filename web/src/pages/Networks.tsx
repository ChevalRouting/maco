import { useMemo, useState } from 'react'
import { EmptyState, PageHeader, Table, TableHeader, TableBody, TableRow, TableHead } from 'maco-ui'
import { Network as NetworkIcon } from 'lucide-react'
import { CreateButton } from '../components/CreateButton'
import { listInterfaces, listNetworks, listVMs } from '../api'
import { buildTopology } from '../lib/topology'
import { NetworkTopology } from '../components/NetworkTopology'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { NetworkSheet } from '../components/NetworkSheet'
import { NetworkRow } from '../components/NetworkRow'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'
import { useSession } from '../hooks/useSession'

export default function Networks() {
  const { admin } = useSession()
  const [creating, setCreating] = useState(false)
  const networks = useResource(listNetworks, [], 'networks')
  const vms = useResource(listVMs, [], 'vms')
  const ports = useResource(listInterfaces, [], 'interfaces')
  const action = useJobAction()
  const topology = useMemo(() => buildTopology(networks.data, vms.data, ports.data), [networks.data, vms.data, ports.data])

  return (
    <div className="space-y-6">
      <PageHeader title="Networks" action={admin ? <CreateButton toolbar label="Create Network…" onClick={() => setCreating(true)} /> : undefined} />
      <ResourceNotice resource={networks} name="networks" />
      <ResourceNotice resource={vms} name="virtual machines" />
      <JobNotice error={action.error} />
      {!networks.loading && (networks.data.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Mode</TableHead>
              <TableHead>Interface</TableHead>
              <TableHead>Address</TableHead>
              <TableHead>VMs</TableHead>
              <TableHead><span className="sr-only">Actions</span></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {networks.data.map((network) => (
              <NetworkRow key={network.id} network={network} vms={vms.data} vmsReady={!vms.loading && !vms.error} run={action.run} />
            ))}
          </TableBody>
        </Table>
      ) : (
        !networks.error && <EmptyState icon={<NetworkIcon />} title="No Networks" action={admin ? <CreateButton label="Create Network…" onClick={() => setCreating(true)} /> : undefined} />
      ))}
      {!networks.loading && <NetworkTopology topology={topology} vmsLoading={vms.loading} />}
      {creating && <NetworkSheet onClose={() => setCreating(false)} />}
    </div>
  )
}
