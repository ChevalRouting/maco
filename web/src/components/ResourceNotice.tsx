import { ResourceNotice as SharedResourceNotice } from 'maco-ui'

interface NoticeResource {
  loading: boolean
  error: string
  refresh: () => void
}

interface ResourceNoticeProps {
  resource: NoticeResource
  name: string
}

export function ResourceNotice({ resource, name }: ResourceNoticeProps) {
  return <SharedResourceNotice name={name} loading={resource.loading} error={resource.error} onRetry={resource.refresh} />
}
