import { useCallback } from 'react'
import { useLocation, useParams } from 'react-router-dom'
import { PageHeader, SectionNav } from 'maco-ui'
import { getVM, listNetworks, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { useTabParam } from '../hooks/useTabParam'
import { useEntity } from '../hooks/useEntity'
import { useJobAction } from '../hooks/useJobAction'
import { useSession } from '../hooks/useSession'
import { useHasParam } from '../hooks/useHasParam'
import { useCompact, useNarrow } from '../hooks/useNarrow'
import { usePageBack } from '../components/usePageBack'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'
import { VMActions } from '../components/VMActions'
import { VMOverview } from '../components/VMOverview'
import { VMUSB } from '../components/VMUSB'
import { VMDisks } from '../components/VMDisks'
import { VMHardware } from '../components/VMHardware'
import { VMGuestSetup } from '../components/VMGuestSetup'
import { VMSnapshots } from '../components/VMSnapshots'
import { VMBackups } from '../components/VMBackups'
import { VMMedia } from '../components/VMMedia'
import { VMNetwork, vmNetworkName } from '../components/VMNetwork'

type VMSection = 'overview' | 'general' | 'guest' | 'network' | 'disks' | 'usb' | 'boot' | 'snapshots' | 'backups'

interface SectionItem {
  key: VMSection
  label: string
  group?: string
}

const sections: SectionItem[] = [
  { key: 'overview', label: 'Overview' },
  { key: 'general', label: 'General', group: 'Configuration' },
  { key: 'guest', label: 'Guest Setup', group: 'Configuration' },
  { key: 'network', label: 'Network', group: 'Configuration' },
  { key: 'disks', label: 'Disks', group: 'Configuration' },
  { key: 'usb', label: 'USB Devices', group: 'Configuration' },
  { key: 'boot', label: 'Boot', group: 'Configuration' },
  { key: 'snapshots', label: 'Snapshots', group: 'Data' },
  { key: 'backups', label: 'Backups', group: 'Data' },
]

interface SectionState {
  sectionEntered?: boolean
}

const configSections = sections.filter((item) => item.key !== 'overview')

interface VMSectionProps {
  section: VMSection
  vm: VMView
  networkName: string
  refresh: () => void
}

function VMSectionContent({ section, vm, networkName, refresh }: VMSectionProps) {
  const action = useJobAction()
  const networks = useResource(listNetworks, [], 'networks')

  const content =
    section === 'general' ? <VMHardware vm={vm} run={action.run} />
    : section === 'guest' ? <VMGuestSetup vm={vm} run={action.run} />
    : section === 'network' ? <VMNetwork vm={vm} networks={networks.data} loading={networks.loading} error={networks.error} run={action.run} />
    : section === 'disks' ? <VMDisks vm={vm} run={action.run} />
    : section === 'usb' ? <VMUSB vm={vm} run={action.run} error={action.error} />
    : section === 'boot' ? <VMMedia vm={vm} onSaved={refresh} />
    : section === 'snapshots' ? <VMSnapshots vm={vm} run={action.run} />
    : section === 'backups' ? <VMBackups vm={vm} run={action.run} />
    : <VMOverview vm={vm} networkName={networkName} />

  return (
    <div className="space-y-4">
      <JobNotice error={section === 'usb' ? '' : action.error} />
      {section === 'network' && <ResourceNotice resource={networks} name="networks" />}
      {content}
    </div>
  )
}

export default function VMDetails() {
  const { admin } = useSession()
  const back = usePageBack('/', 'Virtual Machines')
  const [requested, select] = useTabParam<VMSection>(sections.map((item) => item.key), 'overview')
  const linked = useHasParam('tab')
  const location = useLocation()
  const narrow = useNarrow()
  const compact = useCompact()
  const entered = linked || (location.state as SectionState | null)?.sectionEntered === true
  const landing = narrow && !entered
  const section = admin ? requested : 'overview'
  const { id = '' } = useParams()
  const load = useCallback(() => getVM(id), [id])
  const vm = useEntity<VMView>(load, 'vms', '/')
  const networks = useResource(listNetworks, [], 'networks')
  const action = useJobAction()
  const manifest = vm.data?.manifest
  const networkName =
    (manifest?.interfaces || [{ network: manifest?.network }])
      .map((nic) => vmNetworkName(nic.network, networks.data, networks.loading))
      .join(', ') || 'None'

  return (
    <div className="space-y-6">
      <PageHeader
        title={manifest?.name || 'Virtual Machine'}
        back={back}
        action={vm.data && admin ? <VMActions vm={vm.data} run={action.run} emphasized compact={compact} /> : undefined}
      />
      <ResourceNotice resource={vm} name="virtual machine" />
      <JobNotice error={action.error} />
      {vm.data &&
        (admin ? (
          <>
            {landing && <VMOverview vm={vm.data} networkName={networkName} />}
            <SectionNav items={narrow ? configSections : sections} active={section} onChange={select} initiallyEntered={linked}>
              {!landing && <VMSectionContent key={section} section={section} vm={vm.data} networkName={networkName} refresh={vm.refresh} />}
            </SectionNav>
          </>
        ) : (
          <VMOverview vm={vm.data} networkName={networkName} />
        ))}
    </div>
  )
}
