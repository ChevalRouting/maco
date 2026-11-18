import { Link } from 'react-router-dom'
import { cn } from 'maco-ui'
import { ChevronRight } from 'lucide-react'
import { memoryLabel } from '../ux'
import { type Job, type VMView } from '../api'
import { useRowLink } from '../hooks/useActivate'
import { SourceName } from './SourceName'
import { VMStateChip } from './StateChips'
import { VMActions } from './VMActions'

interface VMListProps {
  vms: VMView[]
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

interface VMListRowProps {
  vm: VMView
  run: VMListProps['run']
}

function VMListRow({ vm, run }: VMListRowProps) {
  const id = vm.manifest.id
  const link = useRowLink(`/vms/${id}`)

  return (
    <li {...link} className={cn('flex min-h-[4.25rem] items-center gap-3 py-2.5 pl-4 pr-3', link.className)}>
      <div className="min-w-0 flex-1">
        <Link to={`/vms/${id}`} className="block truncate text-base font-semibold text-foreground">
          {vm.manifest.name}
        </Link>
        <div className="mt-0.5 flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
          <VMStateChip phase={vm.phase} />
        </div>
        <p className="mt-0.5 truncate text-xs text-muted-foreground">
          <SourceName image={vm.manifest.image} /> · {vm.manifest.cpus} vCPU · {memoryLabel(vm.manifest.memory_mib)}
        </p>
      </div>
      <VMActions vm={vm} run={run} compact />
      <ChevronRight aria-hidden="true" strokeWidth={2.5} className="h-3.5 w-3.5 shrink-0 text-muted-foreground/50" />
    </li>
  )
}

export function VMList({ vms, run }: VMListProps) {
  return (
    <ul aria-label="Virtual Machines" className="inset-separators overflow-hidden rounded-xl bg-card [box-shadow:var(--card-shadow)]">
      {vms.map((vm) => (
        <VMListRow key={vm.manifest.id} vm={vm} run={run} />
      ))}
    </ul>
  )
}
