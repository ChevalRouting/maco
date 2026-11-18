import { useState } from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from 'maco-ui'
import { type VMView } from '../api'
import { UtilizationCharts } from './UtilizationCharts'

interface DashboardVMStatsProps {
  vms: VMView[]
}

export function DashboardVMStats({ vms }: DashboardVMStatsProps) {
  const [selected, setSelected] = useState('all')
  const target = selected === 'all' || vms.some((vm) => vm.manifest.id === selected) ? selected : 'all'

  return (
    <section className="space-y-2" aria-labelledby="vm-usage-title">
      <div className="flex flex-wrap items-end justify-between gap-3 px-1">
        <h2 id="vm-usage-title" className="text-sm font-semibold">Virtual Machines</h2>
        <Select value={target} onValueChange={setSelected}>
          <SelectTrigger aria-label="Virtual Machine" className="w-56"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All virtual machines</SelectItem>
            {vms.map((candidate) => <SelectItem key={candidate.manifest.id} value={candidate.manifest.id}>{candidate.manifest.name}</SelectItem>)}
          </SelectContent>
        </Select>
      </div>
      <UtilizationCharts scope="vm" target={target} />
    </section>
  )
}

