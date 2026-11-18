import { useState } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { TableRow, TableCell, AlertDialog } from 'maco-ui'
import { NetworkSheet } from './NetworkSheet'
import { ActionButton } from './ActionButton'
import { useRowAction } from '../hooks/useRowAction'
import { networkModeName } from '../ux'
import { useSession } from '../hooks/useSession'
import { activatable } from '../hooks/useActivate'
import { destroyNetwork, type Network, type Job, type VMView } from '../api'

interface NetworkRowProps {
  network: Network
  vms: VMView[]
  vmsReady: boolean
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

function target(network: Network): string {
  if (network.mode === 'user') return 'Per adapter'
  if (network.mode === 'vlan') return network.parent || 'None'
  return network.device || network.uplink || network.group || 'Not applied'
}

function address(network: Network): string {
  if (network.mode === 'vlan') return `VLAN ${network.tag}`
  return network.address || 'None'
}

export function NetworkRow({ network, vms, vmsReady, run }: NetworkRowProps) {
  const { admin } = useSession()
  const users = vms.filter((vm) =>
    (vm.manifest.interfaces || [{ network: vm.manifest.network || 'user' }]).some(
      (nic) => nic.network === network.id || nic.network === network.name,
    ),
  )
  const [editing, setEditing] = useState(false)
  const [confirm, setConfirm] = useState(false)
  const action = useRowAction(network.id, network.name, 'network', run)
  const inUse = users.length > 0
  const activation = activatable(admin && !action.busy ? () => setEditing(true) : undefined)

  async function destroy() {
    const job = await action.execute('network.destroy', () => destroyNetwork(network.id))
    if (job) setConfirm(false)
  }

  return (
    <TableRow {...activation}>
      <TableCell data-title className="font-medium">{network.name}</TableCell>
      <TableCell>{networkModeName(network.mode)}</TableCell>
      <TableCell className="font-mono">{target(network)}</TableCell>
      <TableCell className="font-mono">{address(network)}</TableCell>
      <TableCell className="tabular-nums">{vmsReady ? users.length : <span className="text-muted-foreground">…</span>}</TableCell>
      <TableCell>
        {admin && (
          <div className="flex items-center gap-1">
            <ActionButton label={`Edit ${network.name}`} icon={Pencil} disabled={action.busy} onClick={() => setEditing(true)} />
            <ActionButton
              label={inUse ? `${network.name} Is in Use` : `Delete ${network.name}`}
              icon={Trash2}
              disabled={action.busy || !vmsReady || inUse}
              busy={action.action === 'network.destroy'}
              onClick={() => setConfirm(true)}
            />
            {editing && <NetworkSheet network={network} vms={vms} onClose={() => setEditing(false)} />}
            <AlertDialog
              open={confirm}
              onCancel={() => setConfirm(false)}
              onConfirm={destroy}
              busy={action.busy}
              destructive
              title={`Delete ${network.name}?`}
              description="Its host bridge or VLAN interface is removed"
              confirmLabel="Delete Network"
            />
          </div>
        )}
      </TableCell>
    </TableRow>
  )
}
