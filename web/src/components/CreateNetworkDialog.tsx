import { IntegerEntryRow } from './IntegerEntryRow'
import { useState, type FormEvent } from 'react'
import { Dialog, Button, PreferencesGroup } from 'cheval-ui'
import {
  createNetwork,
  updateNetwork,
  listInterfaces,
  listNetworks,
  type CreateNetworkParams,
  type Network,
  type NetworkInterface,
} from '../api'
import { useJobAction } from '../hooks/useJobAction'
import { ChoiceRow } from './ChoiceRow'
import { InterfaceMembers } from './InterfaceMembers'
import { useResource } from '../hooks/useResource'
import { ResourceNotice } from './ResourceNotice'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { cidrError, networkModes, networkModeName, nameError } from '../ux'
import { NoticeBanner } from 'cheval-ui'
import { JobNotice } from './JobNotice'

interface CreateNetworkDialogProps {
  network?: Network
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

export function CreateNetworkDialog({ network, onClose }: CreateNetworkDialogProps) {
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

  return (
    <Dialog
      open
      onClose={close}
      title={network ? `Edit ${network.name}` : "Create network"}
      description="Define connectivity for your virtual machines."
      className="max-w-2xl"
      footer={
        <>
          <Button variant="outline" onClick={close} disabled={action.busy}>
            Cancel
          </Button>
          <Button
            variant="suggested"
            type="submit"
            form="create-network"
            disabled={blocked || !!validation || !form.name.trim() || (form.mode === 'vmnet-bridged' && !form.uplink) || (form.mode === 'vlan' && !parent)}
          >
            {action.busy ? 'Saving…' : network ? 'Save and apply' : 'Create'}
          </Button>
        </>
      }
    >
      <form id="create-network" onSubmit={submit} className="space-y-4">
        <JobNotice error={action.error} />
        {needsPorts && (
          <>
            <ResourceNotice resource={ports} name="host adapters" />
            <ResourceNotice resource={inventory} name="networks" />
          </>
        )}
        {network && (
          <NoticeBanner intent="warning">
            Applying changes updates host connectivity. VMs using this network can lose connections;
            reconnect or update guest addressing afterward.
          </NoticeBanner>
        )}
        <fieldset disabled={blocked}>
        <PreferencesGroup title="Network">
          <ValidatedEntryRow
            error={form.name ? nameError(form.name) : undefined}
            help="Use 1–63 letters, digits, dots, underscores or hyphens; begin with a letter or digit."
            id="network-name"
            title="Name"
            required
            disabled={!!network}
            autoFocus
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          {!network ? <ChoiceRow
            title="Connectivity"
            value={form.mode}
            choices={networkModes}
            help={networkModes.find(item => item.value === form.mode)?.help}
            onChange={(mode) => setForm({ ...form, mode: mode as NetworkForm['mode'] })}
          /> : <p className="px-4 py-2 text-sm text-muted-foreground">Connectivity: {networkModeName(network.mode)}</p>}
          {form.mode === 'bridge' && (
            <>
              <ValidatedEntryRow
                error={cidrError(form.address, true)}
                help="Optional IPv4 address for this Mac on the bridge, for example 192.168.1.20/24. This does not assign guest addresses."
                id="network-address"
                title="Host Address"
                value={form.address}
                onChange={(e) => setForm({ ...form, address: e.target.value })}
              />
              <InterfaceMembers
                title="Bridge members (optional)"
                interfaces={memberOptions}
                selected={members}
                onChange={setMembers}
              />
            </>
          )}
          {form.mode === 'vmnet-bridged' && (
            <ChoiceRow
              title="Physical Adapter"
              value={form.uplink}
              choices={interfaceChoices}
              help={
                !ports.loading && !ports.error && !interfaceChoices.length
                  ? 'No eligible physical adapters found. Connect an adapter or choose another connectivity type.'
                  : 'Choose the local network adapter that will carry VM traffic.'
              }
              onChange={(uplink) => setForm({ ...form, uplink })}
            />
          )}
          {form.mode === 'vlan' && (
            <>
              <ChoiceRow title="Parent interface" value={parent} choices={interfaceChoices} onChange={setParent} />
              <IntegerEntryRow
                id="network-tag"
                title="VLAN tag"
                min={1}
                max={4094}
                required
                value={tag}
                onValueChange={setTag}
              />
              <p className="px-4 py-2 text-xs text-muted-foreground">
                Applying creates a tagged VLAN interface on {parent || 'the selected parent'}. Add it
                to a bridge’s members to attach VMs.
              </p>
            </>
          )}
        </PreferencesGroup>
        </fieldset>
      </form>
    </Dialog>
  )
}
