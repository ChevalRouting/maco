import { Cpu, MemoryStick } from 'lucide-react'
import { memoryLabel, plural } from '../ux'
import { type HostInfo, type StorageStats, type VMView } from '../api'
import { StatCard } from './StatCard'
import { DiskStatCard } from './DiskStatCard'

interface HostStatCardsProps {
  host: HostInfo
  vms: VMView[]
  storage: StorageStats | null
}

export function HostStatCards({ host, vms, storage }: HostStatCardsProps) {
  const running = vms.filter((vm) => vm.phase === 'running')
  const usedCpus = running.reduce((sum, vm) => sum + vm.manifest.cpus, 0)
  const usedMem = running.reduce((sum, vm) => sum + vm.manifest.memory_mib, 0)
  const allocCpus = vms.reduce((sum, vm) => sum + vm.manifest.cpus, 0)
  const allocMem = vms.reduce((sum, vm) => sum + vm.manifest.memory_mib, 0)

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <StatCard
        icon={Cpu}
        label="vCPUs Assigned"
        value={`${usedCpus} of ${host.cpus}`}
        caption={`${plural(running.length, 'running VM')}`}
        meta={`${allocCpus} configured in all VMs`}
      />
      <StatCard
        icon={MemoryStick}
        label="Memory Assigned"
        value={memoryLabel(usedMem)}
        caption={`Host has ${memoryLabel(host.memory_bytes / 1048576)}`}
        meta={`${memoryLabel(allocMem)} configured in all VMs`}
      />
      {storage && <DiskStatCard stats={storage} />}
    </div>
  )
}
