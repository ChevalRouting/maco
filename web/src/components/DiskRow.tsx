import { Link } from 'react-router-dom'
import { Badge, TableRow, TableCell, fmtBytes } from 'cheval-ui'
import { type DiskView } from '../api'

export function DiskRow({ disk }: { disk: DiskView }) {
  return (
    <TableRow>
      <TableCell
        data-label="Disk"
        className="max-w-[16rem] whitespace-normal break-words font-medium"
        title={disk.name}
      >
        {disk.name}
        {disk.boot_first && <Badge variant="neutral" className="ml-2">Boot first</Badge>}
      </TableCell>
      <TableCell data-label="Virtual machine">
        {disk.vm_name ? (
          <Link
            className="text-primary hover:underline"
            to={`/vms/${encodeURIComponent(disk.vm_id)}`}
          >
            {disk.vm_name}
          </Link>
        ) : (
          <span className="text-muted-foreground">
            {disk.orphaned ? 'Not attached to a VM' : 'VM unavailable'}
          </span>
        )}
      </TableCell>
      <TableCell data-label="File Size &amp; Capacity" className="tabular-nums">
        <span className="block">File: {fmtBytes(disk.size_bytes)}</span>
        {disk.capacity_gib > 0 && (
          <span className="block text-muted-foreground">Capacity: {disk.capacity_gib} GiB</span>
        )}
      </TableCell>
      <TableCell data-label="Status">
        {disk.orphaned ? (
          <Badge variant="warning">Not Attached</Badge>
        ) : (
          <Badge variant="neutral">Attached</Badge>
        )}
      </TableCell>
    </TableRow>
  )
}
