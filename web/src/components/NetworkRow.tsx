import { useState } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { CreateNetworkDialog } from './CreateNetworkDialog'
import { ActionButton } from './ActionButton'
import { ActionStatus } from './ActionStatus'
import { useRowAction } from '../hooks/useRowAction'
import { networkModeName } from '../ux'
import { useResource } from '../hooks/useResource'
import { useSession } from '../hooks/useSession'
import { TableRow, TableCell, AlertDialog } from 'cheval-ui'
import { destroyNetwork, listVMs, type Network, type Job } from '../api'

interface NetworkRowProps {
  network: Network
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function NetworkRow({ network, run }: NetworkRowProps) {
  const { admin } = useSession()
  const vms = useResource(listVMs, [], 'vms')
  const users = vms.data.filter((vm) =>
    (vm.manifest.interfaces || [{ network: vm.manifest.network || 'user' }]).some(
      (nic) => nic.network === network.id || nic.network === network.name,
    ),
  )
  const [editing, setEditing] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const action = useRowAction(network.id, network.name, 'network', run)

  async function destroy() {
    const job = await action.execute('network.destroy', () =>
      destroyNetwork(network.id),
    )
    if (job) setConfirm(false)
  }

  return (
    <TableRow>
      <TableCell data-label="Name">{network.name}</TableCell>
      <TableCell data-label="Mode">{networkModeName(network.mode)}</TableCell>
      <TableCell data-label="Target">
        {network.mode === 'user' ? 'Per-interface NAT' : network.mode === 'vlan'
          ? network.parent
          : network.device || network.uplink || network.group || 'Unapplied'}
      </TableCell>
      <TableCell data-label="Address">
        {network.mode === 'vlan'
          ? `VLAN ${network.tag}${network.device ? ` · ${network.device}` : ''}`
          : network.address || 'None'}
      </TableCell>
      <TableCell data-label="Actions">
        {admin ? (
          <>
            <div className="flex flex-wrap items-center gap-1">
              <ActionButton label={`Edit ${network.name}`} icon={Pencil} disabled={action.busy} onClick={() => setEditing(true)} />
              <ActionButton
                label={`Delete ${network.name}`}
                icon={Trash2}
                disabled={action.busy || vms.loading || !!vms.error || users.length > 0}
                busy={action.action === 'network.destroy'}
                onClick={() => setConfirm(true)}
              />
              <ActionStatus job={action.job} error={action.error} />
              {users.length > 0 && (
                <p className="max-w-sm break-words text-sm text-muted-foreground">
                  Used by one or more virtual machines. Move or remove those adapters before deleting.
                </p>
              )}
              {vms.error && (
                <p className="text-sm text-muted-foreground">
                  Couldn’t check network use.{' '}
                  <button className="text-primary underline" onClick={vms.refresh}>
                    Retry
                  </button>
                </p>
              )}
            </div>
            {editing && <CreateNetworkDialog network={network} onClose={() => setEditing(false)} />}
            <AlertDialog
              open={confirm}
              onCancel={() => setConfirm(false)}
              onConfirm={destroy}
              busy={action.busy}
              destructive
              title={`Delete ${network.name}?`}
              description="Delete this network configuration and the host bridge or VLAN resources it owns. Guest disks are retained. Host connectivity through these resources ends."
              confirmLabel="Delete Network"
            />
          </>
        ) : (
          <span className="text-sm text-muted-foreground">Read only</span>
        )}
      </TableCell>
    </TableRow>
  )
}
