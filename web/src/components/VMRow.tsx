import { useCallback, useState } from 'react'
import { Link } from 'react-router-dom'
import { TableRow, TableCell, type ContextMenuPoint } from 'maco-ui'
import { memoryLabel } from '../ux'
import { SourceName } from './SourceName'
import { VMStateChip } from './StateChips'
import { VMPreview } from './VMPreview'
import { VMActions } from './VMActions'
import { type VMView, type Job } from '../api'
import { useRowLink } from '../hooks/useActivate'
import { useSession } from '../hooks/useSession'

interface VMRowProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMRow({ vm, run }: VMRowProps) {
  const id = vm.manifest.id
  const link = useRowLink(`/vms/${id}`)
  const { admin } = useSession()
  const [point, setPoint] = useState<ContextMenuPoint | null>(null)
  const closeContext = useCallback(() => setPoint(null), [])

  function openContext(event: React.MouseEvent) {
    event.preventDefault()
    setPoint({ x: event.clientX, y: event.clientY })
  }

  return (
    <TableRow {...link} onContextMenu={admin ? openContext : undefined}>
      <TableCell className="hidden sm:table-cell">
        <Link to={`/vms/${id}`} aria-label={`Open ${vm.manifest.name}`} tabIndex={-1}>
          <VMPreview id={id} name={vm.manifest.name} running={vm.phase === 'running'} bootTime={vm.boot_time} />
        </Link>
      </TableCell>
      <TableCell data-title className="max-w-[14rem]">
        <Link className="block whitespace-normal break-words font-medium text-primary hover:underline" to={`/vms/${id}`}>
          {vm.manifest.name}
        </Link>
      </TableCell>
      <TableCell>
        <VMStateChip phase={vm.phase} />
      </TableCell>
      <TableCell className="max-w-[16rem] whitespace-normal break-words">
        <SourceName image={vm.manifest.image} />
      </TableCell>
      <TableCell className="tabular-nums">{vm.manifest.cpus}</TableCell>
      <TableCell className="tabular-nums">{memoryLabel(vm.manifest.memory_mib)}</TableCell>
      <TableCell>
        <VMActions vm={vm} run={run} contextPoint={point} onContextClose={closeContext} />
      </TableCell>
    </TableRow>
  )
}
