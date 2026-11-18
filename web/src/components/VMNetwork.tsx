import { useState, type FormEvent } from 'react'
import { AlertDialog, Button, EntryRow, PreferencesGroup, Row, Sheet } from 'maco-ui'
import { Pencil, Plus, Shuffle, Trash2 } from 'lucide-react'
import { addVMInterface, updateVMInterface, removeVMInterface, suggestMAC, type Job, type Network, type VMInterface, type VMView } from '../api'
import { builtinNetworks, macError, networkHelp, networkModeName } from '../ux'
import { useRowAction } from '../hooks/useRowAction'
import { ActionButton } from './ActionButton'
import { ChoiceRow, type Choice } from './ChoiceRow'
import { JobNotice } from './JobNotice'
import { activatable } from '../hooks/useActivate'
import { StableWidth } from './StableWidth'

export function vmNetworkName(ref: string | undefined, networks: Network[], loading = false): string {
  const id = ref || 'user'
  return (
    builtinNetworks.find((item) => item.value === id)?.label ||
    networks.find((network) => network.id === id || network.name === id)?.name ||
    (loading ? 'Loading…' : `Unavailable (${id})`)
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

interface AdapterSheetProps {
  vm: VMView
  adapter: VMInterface | null
  index: number
  networks: Network[]
  loading: boolean
  busy: boolean
  error: string
  onClose: () => void
  onSave: (network: string, mac: string) => void
}

function AdapterSheet({ vm, adapter, index, networks, loading, busy, error, onClose, onSave }: AdapterSheetProps) {
  const [network, setNetwork] = useState(adapter?.network || 'user')
  const [mac, setMAC] = useState(adapter?.mac || '')
  const choices = vmNetworkChoices(networks, network, loading)
  const valid = !loading && !macError(mac) && choices.some((item) => item.value === network && !item.disabled)
  const unchanged = !!adapter && network === (adapter.network || 'user') && mac.trim() === (adapter.mac || '')
  const running = vm.phase === 'running'

  async function suggest() {
    const suggestion = await suggestMAC()
    setMAC(suggestion.mac)
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    if (valid && !unchanged) onSave(network, mac.trim())
  }

  return (
    <Sheet open onClose={() => !busy && onClose()} title={adapter ? `Edit Adapter ${index + 1}` : 'Add Adapter'}>
      <form onSubmit={submit} className="space-y-5">
        <JobNotice error={error} />
        <PreferencesGroup description={running && adapter ? 'Saving interrupts this adapter’s connections' : undefined}>
          <ChoiceRow title="Network" value={network} choices={choices} help={networkHelp(network, networks)} disabled={busy || loading} onChange={setNetwork} />
          <EntryRow
            id="interface-mac"
            title="MAC Address"
            placeholder="Automatic"
            className="font-mono"
            value={mac}
            error={macError(mac)}
            disabled={busy}
            onChange={(event) => setMAC(event.target.value)}
            suffix={<ActionButton label="Suggest MAC Address" icon={Shuffle} disabled={busy} onClick={() => void suggest()} />}
          />
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={busy || !valid || unchanged}>
            <StableWidth value={busy ? 'Saving…' : adapter ? 'Save' : 'Add'} values={[adapter ? 'Save' : 'Add', 'Saving…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
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
  const interfaces = vm.manifest.interfaces || [{ id: 'net0', network: vm.manifest.network || 'user' }]
  const running = vm.phase === 'running'
  const editingIndex = editing && editing !== 'new' ? interfaces.findIndex((item) => item.id === editing.id) : interfaces.length

  async function save(network: string, mac: string) {
    if (!editing) return
    const params = { network, mac }
    const job = await action.execute(editing === 'new' ? 'vm.interface.add' : 'vm.interface.update', () =>
      editing === 'new' ? addVMInterface(vm.manifest.id, params) : updateVMInterface(vm.manifest.id, editing.id, params),
    )
    if (job) setEditing(null)
  }

  async function remove() {
    if (!removing) return
    const job = await action.execute('vm.interface.remove', () => removeVMInterface(vm.manifest.id, removing.id))
    if (job) setRemoving(null)
  }

  return (
    <div className="max-w-3xl space-y-4">
      <JobNotice error={error} />
      <PreferencesGroup
        title="Adapters"
        header={
          <Button variant="outline" className="gap-2" disabled={action.busy || loading || !!error || interfaces.length >= 32} onClick={() => setEditing('new')}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Add Adapter…
          </Button>
        }
      >
        {interfaces.length === 0 && <Row subtitle="None" />}
        {interfaces.map((nic, index) => (
          <div key={nic.id} {...activatable(action.busy ? undefined : () => setEditing(nic))}>
            <Row
              title={`Adapter ${index + 1}`}
              subtitle={
                <>
                  {vmNetworkName(nic.network, networks, loading)} · <span className="font-mono">{nic.mac || 'automatic MAC'}</span> · VirtIO
                </>
              }
            >
              <ActionButton label={`Edit Adapter ${index + 1}`} icon={Pencil} disabled={action.busy} onClick={() => setEditing(nic)} />
              <ActionButton label={`Remove Adapter ${index + 1}`} icon={Trash2} disabled={action.busy} onClick={() => setRemoving(nic)} />
            </Row>
          </div>
        ))}
      </PreferencesGroup>

      {editing && (
        <AdapterSheet
          vm={vm}
          adapter={editing === 'new' ? null : editing}
          index={editingIndex}
          networks={networks}
          loading={loading}
          busy={action.busy}
          error={action.error}
          onClose={() => setEditing(null)}
          onSave={(network, mac) => void save(network, mac)}
        />
      )}
      <AlertDialog
        open={!!removing}
        onCancel={() => setRemoving(null)}
        onConfirm={remove}
        busy={action.busy}
        destructive
        title={`Remove Adapter ${removing ? interfaces.findIndex((item) => item.id === removing.id) + 1 : ''}?`}
        description={running ? 'Its connections are interrupted immediately' : undefined}
        confirmLabel="Remove Adapter"
      />
    </div>
  )
}
