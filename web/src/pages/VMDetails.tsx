import { useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { PageHeader, Tabs } from 'cheval-ui'
import { getVM, listNetworks, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { useTabParam } from '../hooks/useTabParam'
import { useEntity } from '../hooks/useEntity'
import { useJobAction } from '../hooks/useJobAction'
import { useSession } from '../hooks/useSession'
import { BackLink } from '../components/BackLink'
import { ResourceNotice } from '../components/ResourceNotice'
import { JobNotice } from '../components/JobNotice'

import { VMUSB } from '../components/VMUSB'
import { VMDisks } from '../components/VMDisks'
import { VMHardware } from '../components/VMHardware'
import { VMGuestSetup } from '../components/VMGuestSetup'
import { VMWorkspace } from '../components/VMWorkspace'
import { VMSnapshotsBackups } from '../components/VMSnapshotsBackups'

import { VMMedia } from '../components/VMMedia'
import { VMNetwork, vmNetworkName } from '../components/VMNetwork'

type VMTab = 'overview' | 'compute' | 'guest' | 'disks' | 'usb' | 'media' | 'network' | 'backups'

const tabs: { key: VMTab; label: string }[] = [
  { key: 'overview', label: 'Overview' },
  { key: 'compute', label: 'CPU & memory' },
  { key: 'guest', label: 'Guest setup' },
  { key: 'network', label: 'Network' },
  { key: 'disks', label: 'Disks' },
  { key: 'usb', label: 'USB devices' },
  { key: 'media', label: 'Media & boot' },
  { key: 'backups', label: 'Snapshots & backups' },
]

export default function VMDetails() {
  const { admin } = useSession()
  const visibleTabs = admin ? tabs : tabs.filter((item) => item.key === 'overview')
  const [requested, selectTab] = useTabParam<VMTab>(tabs.map((item) => item.key), 'overview')
  const tab = admin ? requested : 'overview'
  const { id = '' } = useParams()
  const load = useCallback(() => getVM(id), [id])
  const vm = useEntity<VMView>(load, 'vms', '/')
  const networks = useResource(listNetworks, [], 'networks')
  const action = useJobAction()
  const manifest = vm.data?.manifest

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-col gap-6">
      <BackLink to="/">Back to virtual machines</BackLink>
      <PageHeader title={manifest?.name || 'Virtual machine'} />
      <ResourceNotice resource={vm} name="virtual machine" />
      <JobNotice error={action.error} />
      {vm.data && (
        <>
          <Tabs id="vm-details" label="VM Settings" tabs={visibleTabs} active={tab} onChange={selectTab} />
          <div className="min-h-0 flex-1 overflow-y-auto">
            {visibleTabs.map((item) => (
              <div
                key={`${id}:${item.key}`}
                role="tabpanel"
                id={`vm-details-panel-${item.key}`}
                aria-labelledby={`vm-details-tab-${item.key}`}
                hidden={tab !== item.key}
              >
                {item.key === 'overview' ? (
                  <VMWorkspace
                    vm={vm.data!}
                    run={action.run}
                    networkName={
                      (manifest?.interfaces || [{ network: manifest?.network }])
                        .map((nic) => vmNetworkName(nic.network, networks.data, networks.loading))
                        .join(', ') || 'None'
                    }
                  />
                ) : item.key === 'media' ? (
                  <VMMedia vm={vm.data!} onSaved={vm.refresh} />
                ) : item.key === 'network' ? (
                  <>
                    <ResourceNotice resource={networks} name="networks" />
                    <VMNetwork
                      vm={vm.data!}
                      networks={networks.data}
                      loading={networks.loading}
                      error={networks.error}
                      run={action.run}
                    />
                  </>
                ) : item.key === 'usb' ? (
                  <VMUSB vm={vm.data!} run={action.run} error={action.error} />
                ) : item.key === 'backups' ? (
                  <VMSnapshotsBackups vm={vm.data!} run={action.run} />
                ) : item.key === 'disks' ? (
                  <VMDisks vm={vm.data!} run={action.run} />
                ) : item.key === 'guest' ? (
                  <VMGuestSetup vm={vm.data!} run={action.run} />
                ) : (
                  <VMHardware vm={vm.data!} run={action.run} />
                )}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  )
}
