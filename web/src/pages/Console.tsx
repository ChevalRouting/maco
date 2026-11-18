import { EmptyState, PageHeader } from 'maco-ui'
import { Lock } from 'lucide-react'
import { HostConsole } from '../components/HostConsole'
import { useSession } from '../hooks/useSession'

export default function Console() {
  const { admin } = useSession()

  return (
    <div className="space-y-6">
      <PageHeader title="Console" />
      {admin ? <HostConsole /> : <EmptyState icon={<Lock />} title="Administrator Required" />}
    </div>
  )
}
