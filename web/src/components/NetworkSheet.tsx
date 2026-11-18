import { useState, type FormEvent } from 'react'
import { Button, PreferencesGroup, Row, Sheet } from 'maco-ui'
import { IntegerEntryRow } from './IntegerEntryRow'
import {
  createNetwork,
  updateNetwork,
  listInterfaces,
  listNetworks,
  type CreateNetworkParams,
  type Network,
  type NetworkInterface,
  type VMView,
} from '../api'
import { topologyRow } from '../lib/topology'
import { SwitchDiagram } from './NetworkTopology'
import { useJobAction } from '../hooks/useJobAction'
import { ChoiceRow } from './ChoiceRow'
import { InterfaceMembers } from './InterfaceMembers'
import { useResource } from '../hooks/useResource'
import { ResourceNotice } from './ResourceNotice'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { cidrError, networkModes, networkModeName, nameError } from '../ux'
import { JobNotice } from './JobNotice'
import { StableWidth } from './StableWidth'

interface NetworkSheetProps {
  network?: Network
  vms?: VMView[]
  onClose: () => void
}

type NetworkForm = Required<Omit<CreateNetworkParams, 'device'>> & Pick<CreateNetworkParams, 'device'>

const defaults: NetworkForm = {
  name: '',
  mode: 'bridge',
  address: '',
  uplink: '',
  parent: '',
  tag: 0,
  members: [],
  vlans: [],
}

export function NetworkSheet({ network, vms, onClose }: NetworkSheetProps) {
  const [form, setForm] = useState<NetworkForm>(
    network
      ? {
          name: network.name,
          mode: network.mode,
          device: network.device,
          address: network.address || '',
          uplink: network.uplink || '',
          parent: network.parent || '',
          tag: network.tag || 0,
          members: network.members || [],
          vlans: network.vlans || [],
        }
      : defaults,
  )
  const [members, setMembers] = useState<string[]>(network?.members || [])
  const [parent, setParent] = useState(network?.parent || network?.vlans?.[0]?.parent || '')
  const [tag, setTag] = useState(network?.tag || network?.vlans?.[0]?.tag || 100)
  const ports = useResource(listInterfaces, [], 'interfaces')
  const inventory = useResource(listNetworks, [], 'networks')
  const interfaces = ports.data.filter(iface => iface.hardware_port)
  const networks = inventory.data
  const action = useJobAction()
  const needsPorts = ['bridge', 'vlan', 'vmnet-bridged'].includes(form.mode)
  const blocked = action.busy || (needsPorts && (ports.loading || !!ports.error || inventory.loading || !!inventory.error))
  const validation = nameError(form.name) || (form.mode === 'bridge' ? cidrError(form.address, true) : '')

  const extraDevices = [...new Set([network?.uplink, parent].filter((value): value is string => !!value))]
    .filter((device) => !interfaces.some((iface) => iface.device === device))
    .map((device) => ({ device, hardware_port: '', up: false, addresses: [] }))
  const availableInterfaces = [...interfaces, ...extraDevices]
  const interfaceChoices = availableInterfaces.map((iface) => ({
    value: iface.device,
    label: iface.hardware_port
      ? `${iface.device} · ${iface.hardware_port}`
      : iface.device,
  }))

  const vlanMembers: NetworkInterface[] = networks
    .filter((network) => network.mode === 'vlan' && network.device)
    .map((network) => ({
      device: network.device as string,
      hardware_port: `VLAN ${network.tag} · ${network.name}`,
      up: true,
      addresses: [],
    }))

  const detectedMembers = [...interfaces, ...vlanMembers]
  const extraMembers = members
    .filter((device) => !detectedMembers.some((iface) => iface.device === device))
    .map((device) => ({ device, hardware_port: '', up: false, addresses: [] }))
  const memberOptions = [...detectedMembers, ...extraMembers]

  async function submit(event: FormEvent) {
    event.preventDefault()
    const params: CreateNetworkParams = {
      ...form,
      uplink: form.mode === 'vmnet-bridged' ? form.uplink : '',
      address: form.mode === 'bridge' ? form.address : '',
      parent: form.mode === 'vlan' ? parent : '',
      tag: form.mode === 'vlan' ? tag : 0,
      members: form.mode === 'bridge' ? members : [],
      vlans: form.mode === 'bridge' ? network?.vlans || [] : [],
    }
    if (blocked || validation) return
    const job = await action.run(() => network ? updateNetwork(network.id, params) : createNetwork(params))
    if (job) {
      onClose()
    }
  }

  function close() {
    if (!action.busy) onClose()
  }

  const submitDisabled =
    blocked || !!validation || !form.name.trim() || (form.mode === 'vmnet-bridged' && !form.uplink) || (form.mode === 'vlan' && !parent)

  return (
    <Sheet open onClose={close} title={network ? `Edit ${network.name}` : 'Create Network'}>
      <form onSubmit={submit} className="space-y-5">
        <JobNotice error={action.error} />
        {needsPorts && (
          <>
            <ResourceNotice resource={ports} name="host adapters" />
            <ResourceNotice resource={inventory} name="networks" />
          </>
        )}
        {network && vms && !inventory.loading && (
          <PreferencesGroup title="Topology">
            <div className="px-4 pb-2 pt-1">
              <SwitchDiagram row={topologyRow(network, networks, vms, ports.data)} />
            </div>
          </PreferencesGroup>
        )}
        <fieldset disabled={blocked} className="space-y-5">
          <PreferencesGroup>
            <ValidatedEntryRow
              error={form.name ? nameError(form.name) : undefined}
              id="network-name"
              title="Name"
              placeholder="lab"
              required
              disabled={!!network}
              autoFocus
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
            {network ? (
              <Row title="Mode">
                <span className="text-sm">{networkModeName(network.mode)}</span>
              </Row>
            ) : (
              <ChoiceRow
                title="Mode"
                value={form.mode}
                choices={networkModes}
                help={networkModes.find((item) => item.value === form.mode)?.help}
                onChange={(mode) => setForm({ ...form, mode: mode as NetworkForm['mode'] })}
              />
            )}
          </PreferencesGroup>
          {form.mode === 'bridge' && (
            <>
            <PreferencesGroup title="Bridge">
              <ValidatedEntryRow
                error={cidrError(form.address, true)}
                id="network-address"
                title="Host Address"
                placeholder="192.168.64.1/24"
                className="font-mono"
                value={form.address}
                onChange={(e) => setForm({ ...form, address: e.target.value })}
              />
            </PreferencesGroup>
            <PreferencesGroup title="Members">
              <InterfaceMembers interfaces={memberOptions} selected={members} onChange={setMembers} />
            </PreferencesGroup>
            </>
          )}
          {form.mode === 'vmnet-bridged' && (
            <PreferencesGroup>
              <ChoiceRow
                title="Physical Adapter"
                value={form.uplink}
                choices={interfaceChoices}
                onChange={(uplink) => setForm({ ...form, uplink })}
              />
            </PreferencesGroup>
          )}
          {form.mode === 'vlan' && (
            <PreferencesGroup title="VLAN">
              <ChoiceRow title="Parent Interface" value={parent} choices={interfaceChoices} onChange={setParent} />
              <IntegerEntryRow id="network-tag" title="VLAN ID" min={1} max={4094} required value={tag} onValueChange={setTag} />
            </PreferencesGroup>
          )}
        </fieldset>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={close} disabled={action.busy}>
            Cancel
          </Button>
          <Button variant="suggested" type="submit" disabled={submitDisabled}>
            <StableWidth value={action.busy ? 'Saving…' : network ? 'Save' : 'Create'} values={[network ? 'Save' : 'Create', 'Saving…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}
