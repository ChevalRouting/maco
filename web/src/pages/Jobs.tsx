import { PageHeader } from 'maco-ui'
import { useJobs } from '../hooks/useJobs'
import { JobsList } from '../components/JobsList'

export default function Jobs() {
  const resource = useJobs()

  return (
    <div className="space-y-6">
      <PageHeader title="Activity" />
      <JobsList resource={resource} />
    </div>
  )
}
