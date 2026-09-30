import { useState } from 'react'
import {
  Button,
  EmptyState,
  PageHeader,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from 'cheval-ui'
import { Network as NetworkIcon, Plus } from 'lucide-react'
import { listNetworks } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { CreateNetworkDialog } from '../components/CreateNetworkDialog'
import { NetworkRow } from '../components/NetworkRow'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'
import { useSession } from '../hooks/useSession'

export default function Networks() {
  const { admin } = useSession()
  const [creating, setCreating] = useState(false)
  const networks = useResource(listNetworks, [], 'networks')
  const action = useJobAction()

  return (
    <div className="space-y-6">
      <PageHeader
        title="Networks"
        action={
          admin ? (
            <Button
              variant="suggested"
              className="gap-2"
              onClick={() => setCreating(true)}
            >
              <Plus className="h-4 w-4 shrink-0" />
              Create network
            </Button>
          ) : undefined
        }
      />
      <ResourceNotice resource={networks} name="networks" />
      <JobNotice error={action.error} />
      {!networks.loading && (networks.data.length ? (
        <Table className="table-cards">
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Mode</TableHead>
              <TableHead>Target</TableHead>
              <TableHead>Address</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {networks.data.map((network) => (
              <NetworkRow key={network.id} network={network} run={action.run} />
            ))}
          </TableBody>
        </Table>
      ) : (
        !networks.error && (
          <EmptyState
            icon={<NetworkIcon />}
            title="No networks yet"
            message="Create a network to define its topology."
          />
        )
      ))}
      {creating && <CreateNetworkDialog onClose={() => setCreating(false)} />}
    </div>
  )
}
