import { memoryLabel } from '../ux'
import { SourceName } from './SourceName'
import { VMStatusBadge } from './StatusBadge'
import { VMPreview } from './VMPreview'
import { Link } from 'react-router-dom'
import { TableRow, TableCell } from 'cheval-ui'
import { VMActions } from './VMActions'
import { type VMView, type Job } from '../api'

interface VMRowProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMRow({ vm, run }: VMRowProps) {
  const id = vm.manifest.id
  const running = vm.phase === 'running'

  return (
    <TableRow>
      <TableCell>
        <Link to={`/vms/${id}`} aria-label={`Open ${vm.manifest.name}`}>
          <VMPreview
            id={id}
            name={vm.manifest.name}
            running={running}
            bootTime={vm.boot_time}
          />
        </Link>
      </TableCell>
      <TableCell data-label="Name" className="max-w-[14rem]">
        <Link
          className="block whitespace-normal break-words font-medium text-primary hover:underline"
          to={`/vms/${id}`}
          title={vm.manifest.name}
        >
          {vm.manifest.name}
        </Link>
      </TableCell>
      <TableCell data-label="Source" className="max-w-[16rem] whitespace-normal break-words">
        <SourceName image={vm.manifest.image} />
      </TableCell>
      <TableCell data-label="Memory">{memoryLabel(vm.manifest.memory_mib)}</TableCell>
      <TableCell data-label="State">
        <VMStatusBadge phase={vm.phase} />
      </TableCell>
      <TableCell data-label="Automatic Startup">{vm.manifest.autostart ? 'Yes' : 'No'}</TableCell>
      <TableCell data-label="Actions">
        <VMActions vm={vm} run={run} />
      </TableCell>
    </TableRow>
  )
}
