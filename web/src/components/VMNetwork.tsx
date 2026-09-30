import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { AlertDialog, Badge, Button, Dialog, NoticeBanner, PreferencesGroup } from 'cheval-ui'
import { Plus } from 'lucide-react'
import {
  addVMInterface,
  updateVMInterface,
  removeVMInterface,
  suggestMAC,
  type Job,
  type Network,
  type VMInterface,
  type VMView,
} from '../api'
import { builtinNetworks, macError, networkHelp, networkModeName } from '../ux'
import { useRowAction } from '../hooks/useRowAction'
import { ChoiceRow, type Choice } from './ChoiceRow'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { ActionStatus } from './ActionStatus'
import { JobNotice } from './JobNotice'

export function vmNetworkName(ref: string | undefined, networks: Network[], loading = false): string {
  const id = ref || 'user'
  return (
    builtinNetworks.find((item) => item.value === id)?.label ||
    networks.find((network) => network.id === id || network.name === id)?.name ||
    (loading ? 'Loading Network…' : `Unavailable Network (${id})`)
  )
}

export function vmNetworkChoices(networks: Network[], selected?: string, loading = false): Choice[] {
  const choices: Choice[] = [
    ...builtinNetworks,
    ...networks
      .filter((item) => item.mode !== 'vlan')
      .map((item) => ({ value: item.id, label: `${item.name} · ${networkModeName(item.mode)}` })),
  ]
  if (selected && !choices.some((item) => item.value === selected)) {
    choices.push({ value: selected, label: vmNetworkName(selected, networks, loading), disabled: true })
  }
  return choices
}

interface VMNetworkProps {
  vm: VMView
  networks: Network[]
  loading: boolean
  error: string
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMNetwork({ vm, networks, loading, error, run }: VMNetworkProps) {
  const action = useRowAction(vm.manifest.id, vm.manifest.name, 'vm', run)
  const [editing, setEditing] = useState<VMInterface | 'new' | null>(null)
  const [removing, setRemoving] = useState<VMInterface | null>(null)
  const [confirmSave, setConfirmSave] = useState(false)
  const [network, setNetwork] = useState('user')
  const [mac, setMAC] = useState('')

  const interfaces = vm.manifest.interfaces || [
    { id: 'net0', network: vm.manifest.network || 'user' },
  ]
  const running = vm.phase === 'running'
  const choices = vmNetworkChoices(networks, network, loading)
  const valid =
    !loading && !error && !macError(mac) && choices.some((item) => item.value === network && !item.disabled)
  const unchanged =
    editing !== 'new' &&
    editing &&
    network === (editing.network || 'user') &&
    mac.trim() === (editing.mac || '')

  function edit(nic: VMInterface | 'new') {
    setNetwork(nic === 'new' ? 'user' : nic.network || 'user')
    setMAC(nic === 'new' ? '' : nic.mac || '')
    setEditing(nic)
  }

  async function suggest() {
    if (action.busy) return
    const suggestion = await suggestMAC()
    setMAC(suggestion.mac)
  }

  async function apply() {
    if (action.busy || !editing || !valid) return
    const params = { network, mac: mac.trim() }
    const job = await action.execute(
      editing === 'new' ? 'vm.interface.add' : 'vm.interface.update',
      () =>
        editing === 'new'
          ? addVMInterface(vm.manifest.id, params)
          : updateVMInterface(vm.manifest.id, editing.id, params),
    )
    if (job) {
      setConfirmSave(false)
      setEditing(null)
    }
  }

  function save(event: FormEvent) {
    event.preventDefault()
    if (!valid || unchanged) return
    if (running && editing !== 'new') setConfirmSave(true)
    else void apply()
  }

  async function remove() {
    if (!removing || action.busy) return
    const job = await action.execute('vm.interface.remove', () =>
      removeVMInterface(vm.manifest.id, removing.id),
    )
    if (job) setRemoving(null)
  }

  return (
    <div className="max-w-3xl space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="font-semibold">Network Adapters ({interfaces.length})</h2>
        <Button
          size="sm"
          variant="suggested"
          disabled={action.busy || loading || !!error || interfaces.length >= 32}
          onClick={() => edit('new')}
        >
          <Plus aria-hidden="true" className="mr-2 h-4 w-4" />
          Add Adapter
        </Button>
      </div>

      <p className="text-sm text-muted-foreground">
        {running
          ? 'Adapters can change while the VM runs. The guest must support device changes; changing a network or MAC interrupts that adapter’s connections.'
          : 'Saved adapters connect when the VM starts.'}{' '}
        maco does not configure addresses inside an existing guest.
      </p>

      <JobNotice error={error} />
      <ActionStatus job={action.job} error={action.error} hideProgress />

      <PreferencesGroup>
        {interfaces.map((nic, index) => (
          <article key={nic.id} className="space-y-3 p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h3 className="font-semibold">Adapter {index + 1}</h3>
              <Badge variant="neutral">VirtIO</Badge>
            </div>
            <p className="break-words font-medium">{vmNetworkName(nic.network, networks, loading)}</p>
            <p className="text-sm text-muted-foreground">{networkHelp(nic.network || 'user', networks)}</p>
            <details className="text-sm text-muted-foreground">
              <summary className="cursor-pointer">Adapter Details</summary>
              <p className="mt-2 break-all">
                ID: {nic.id} · MAC: {nic.mac || 'Assigned Automatically'}
              </p>
            </details>
            <div className="flex gap-2">
              <Button size="sm" disabled={action.busy} onClick={() => edit(nic)}>
                Edit…
              </Button>
              <Button size="sm" variant="outline" disabled={action.busy} onClick={() => setRemoving(nic)}>
                Remove…
              </Button>
            </div>
          </article>
        ))}
      </PreferencesGroup>

      {!interfaces.length && (
        <p className="text-sm text-muted-foreground">
          No network adapters attached. This VM has no network connectivity.
        </p>
      )}

      <Link to="/networks" className="inline-block text-sm text-primary underline">
        Manage Networks
      </Link>

      {editing && (
        <Dialog
          open
          title={
            editing === 'new'
              ? 'Add Network Adapter'
              : `Edit Adapter ${interfaces.findIndex((item) => item.id === editing.id) + 1}`
          }
          onClose={() => {
            if (!action.busy) setEditing(null)
          }}
        >
          <form onSubmit={save} className="space-y-4">
            <PreferencesGroup title="Connectivity">
              <ChoiceRow
                title="Network"
                value={network}
                choices={choices}
                help={networkHelp(network, networks)}
                disabled={action.busy || loading}
                onChange={setNetwork}
              />
            </PreferencesGroup>
            <details>
              <summary className="cursor-pointer text-sm font-medium">Advanced Options</summary>
              <PreferencesGroup className="mt-3">
                <ValidatedEntryRow
                  id="interface-mac"
                  title="MAC Address"
                  placeholder="Assigned Automatically"
                  value={mac}
                  error={macError(mac)}
                  help="Leave empty for automatic assignment. A custom address must be unique on this network."
                  disabled={action.busy}
                  onChange={(event) => setMAC(event.target.value)}
                />
              </PreferencesGroup>
              <div className="mt-2 flex gap-2">
                <Button type="button" size="sm" variant="outline" disabled={action.busy} onClick={() => void suggest()}>
                  Suggest MAC
                </Button>
                {mac.trim() && (
                  <Button type="button" size="sm" variant="ghost" disabled={action.busy} onClick={() => setMAC('')}>
                    Use Automatic
                  </Button>
                )}
              </div>
            </details>
            {running && editing !== 'new' && (
              <NoticeBanner intent="warning">
                Applying changes can disconnect this VM, including SSH sessions. Reconnect using the
                guest’s current address afterward.
              </NoticeBanner>
            )}
            <div className="flex justify-end gap-2">
              <Button type="button" disabled={action.busy} onClick={() => setEditing(null)}>
                Cancel
              </Button>
              <Button type="submit" variant="suggested" disabled={action.busy || !valid || !!unchanged}>
                {action.busy ? 'Saving…' : editing === 'new' ? 'Add Adapter' : 'Save & Apply'}
              </Button>
            </div>
          </form>
        </Dialog>
      )}

      <AlertDialog
        open={confirmSave}
        onCancel={() => setConfirmSave(false)}
        onConfirm={apply}
        busy={action.busy}
        title="Apply Network Changes?"
        description="This running VM can lose existing connections on the adapter. The guest may need a new address or network configuration."
        confirmLabel="Apply Changes"
      />
      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title="Remove Network Adapter?"
        description={
          running
            ? 'This immediately disconnects the adapter and its active connections. Other adapters remain connected. You can add a new adapter later.'
            : 'This adapter will no longer connect when the VM starts. You can add a new adapter later.'
        }
        confirmLabel="Remove Adapter"
      />
    </div>
  )
}
