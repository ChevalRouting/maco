import { Link } from 'react-router-dom'
import { activatable, useRowLink } from '../hooks/useActivate'
import { Badge, StatusChip, TableRow, TableCell, fmtBytes } from 'maco-ui'
import { type DiskView } from '../api'

interface DiskRowProps {
  disk: DiskView
}

export function DiskRow({ disk }: DiskRowProps) {
  const link = useRowLink(`/vms/${encodeURIComponent(disk.vm_id)}?tab=disks`)
  const none = activatable()

  return (
    <TableRow {...(disk.vm_name ? link : none)}>
      <TableCell data-title className="max-w-[16rem] whitespace-normal break-words font-medium">
        {disk.name}
        {disk.boot_first && <Badge variant="neutral" className="ml-2">Boot</Badge>}
      </TableCell>
      <TableCell>
        {disk.vm_name ? (
          <Link className="text-primary hover:underline" to={`/vms/${encodeURIComponent(disk.vm_id)}`}>
            {disk.vm_name}
          </Link>
        ) : (
          <span className="text-muted-foreground">None</span>
        )}
      </TableCell>
      <TableCell className="tabular-nums">{fmtBytes(disk.size_bytes)}</TableCell>
      <TableCell className="tabular-nums">{disk.capacity_gib > 0 ? `${disk.capacity_gib} GiB` : 'Unknown'}</TableCell>
      <TableCell>
        <StatusChip intent={disk.orphaned ? 'warning' : 'neutral'} label={disk.orphaned ? 'Detached' : 'Attached'} />
      </TableCell>
    </TableRow>
  )
}
