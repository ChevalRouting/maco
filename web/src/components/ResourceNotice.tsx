import { ResourceNotice as SharedResourceNotice } from 'cheval-ui'

interface ResourceNoticeProps {
  resource: { loading: boolean; error: string; refresh: () => void }
  name: string
}

export function ResourceNotice({ resource, name }: ResourceNoticeProps) {
  return <SharedResourceNotice name={name} loading={resource.loading} error={resource.error} onRetry={resource.refresh} />
}
