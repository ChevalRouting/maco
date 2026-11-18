import { PageHeader } from 'maco-ui'
import { getDiskStorage, getHost, listVMs } from '../api'
import { DashboardVMStats } from '../components/DashboardVMStats'
import { HostStatCards } from '../components/HostStatCards'
import { ResourceNotice } from '../components/ResourceNotice'
import { UtilizationCharts } from '../components/UtilizationCharts'
import { useResource } from '../hooks/useResource'

export default function Dashboard() {
  const host = useResource(getHost, null, 'host')
  const vms = useResource(listVMs, [], 'vms')
  const storage = useResource(getDiskStorage, null, 'storage')

  return (
    <div className="space-y-6">
      <PageHeader title="Dashboard" />
      <ResourceNotice resource={host} name="host" />
      <ResourceNotice resource={storage} name="host storage" />
      <ResourceNotice resource={vms} name="virtual machines" />
      {host.data && <HostStatCards host={host.data} vms={vms.data} storage={storage.data} />}
      <section className="space-y-2" aria-labelledby="host-usage-title">
        <h2 id="host-usage-title" className="px-1 text-sm font-semibold">Host</h2>
        <UtilizationCharts scope="host" target="host" />
      </section>
      <DashboardVMStats vms={vms.data} />
    </div>
  )
}
