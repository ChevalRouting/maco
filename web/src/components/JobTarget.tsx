import { Link } from 'react-router-dom'
import { listVMs, type Job } from '../api'
import { useResource } from '../hooks/useResource'

interface JobTargetProps {
  job: Job
}

export function JobTarget({ job }: JobTargetProps) {
  const vms = useResource(listVMs, [], 'vms')
  const target = job.action === 'vm.create' ? job.result : job.target
  const vm =
    job.action.startsWith('vm.') &&
    vms.data.find((item) => item.manifest.id === target || item.manifest.name === target)

  if (!vm) return <span className="break-words">{job.target || 'Not Recorded'}</span>

  return (
    <Link
      className="whitespace-normal break-words text-primary hover:underline"
      to={`/vms/${encodeURIComponent(vm.manifest.id)}`}
    >
      {vm.manifest.name}
    </Link>
  )
}
