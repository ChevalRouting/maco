import { HardDrive } from 'lucide-react'
import { fmtBytes } from 'maco-ui'
import { type StorageStats } from '../api'
import { plural } from '../ux'
import { StatCard } from './StatCard'

interface DiskStatCardProps {
  stats: StorageStats
}

export function DiskStatCard({ stats }: DiskStatCardProps) {
  const pct = stats.total_bytes > 0 ? ((stats.total_bytes - stats.free_bytes) / stats.total_bytes) * 100 : 0

  return (
    <StatCard
      icon={HardDrive}
      label="Host Storage"
      value={`${fmtBytes(stats.free_bytes)} free`}
      caption={`${fmtBytes(stats.total_bytes)} volume`}
      pct={pct}
      barLabel="Volume used"
      warn={pct >= 90}
      meta={`VM disks: ${fmtBytes(stats.disks_bytes)} in ${plural(stats.disk_count, 'disk')}`}
    />
  )
}
