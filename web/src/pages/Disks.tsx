import { useCallback, useState } from 'react'
import {
  EmptyState,
  PageHeader,
  Pagination,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
} from 'cheval-ui'
import { HardDrive } from 'lucide-react'
import { listDisks, getDiskStorage, type DiskPage } from '../api'
import { useResource } from '../hooks/useResource'
import { DiskRow } from '../components/DiskRow'
import { DiskStatCard } from '../components/DiskStatCard'
import { ResourceNotice } from '../components/ResourceNotice'

const PAGE_SIZE = 25

export default function Disks() {
  const [page, setPage] = useState(1)
  const load = useCallback(() => listDisks(page), [page])
  const disks = useResource<DiskPage | null>(load, null, 'disks')
  const storage = useResource(getDiskStorage, null, 'storage')

  const items = disks.data?.items || []
  const total = disks.data?.total || 0
  const totalPages = disks.data?.total_pages || 1
  const current = disks.data?.page || 1
  const loading = disks.loading && !disks.data

  return (
    <div className="space-y-6">
      <PageHeader title="Disks" />
      <ResourceNotice resource={storage} name="host storage" />
      {storage.data && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <DiskStatCard stats={storage.data} />
        </div>
      )}

      <ResourceNotice resource={disks} name="disk inventory" />

      {!loading && (
        total ? (
          <>
            <Table className="table-cards">
              <TableHeader>
                <TableRow>
                  <TableHead>Disk</TableHead>
                  <TableHead>Virtual machine</TableHead>
                  <TableHead>File Size &amp; Capacity</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((disk) => (
                  <DiskRow key={disk.path} disk={disk} />
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={current - 1}
              totalPages={totalPages}
              total={total}
              pageSize={PAGE_SIZE}
              onPage={(value) => setPage(value + 1)}
              unit="disks"
            />
          </>
        ) : (
          !disks.error && (
            <EmptyState
              icon={<HardDrive />}
              title="No disks yet"
              message="Disks appear here once you create virtual machines."
            />
          )
        )
      )}
    </div>
  )
}
