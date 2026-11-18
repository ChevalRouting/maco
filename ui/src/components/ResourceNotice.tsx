import { Button } from '@/components/ui/button'
import { NoticeBanner } from '@/components/NoticeBanner'
import { Spinner } from '@/components/Spinner'

export interface ResourceNoticeProps {
  loading?: boolean
  error?: string | null
  name: string
  onRetry: () => void
}

export function ResourceNotice({ loading, error, name, onRetry }: ResourceNoticeProps) {
  if (loading) {
    return (
      <div
        role="status"
        className="flex items-center gap-2 text-sm text-muted-foreground"
      >
        <Spinner
          size="sm"
          className="h-auto"
        />
        {`Loading ${name}…`}
      </div>
    )
  }
  if (!error) {
    return null
  }

  return (
    <NoticeBanner
      intent="danger"
    >
      <div
        className="space-y-2"
      >
        <p>
          {`Could not refresh ${name}. Previously loaded information may be out of date.`}
        </p>
        <Button
          type="button"
          onClick={onRetry}
        >
          Retry
        </Button>
        <details>
          <summary
            className="cursor-pointer text-xs"
          >
            Technical details
          </summary>
          <p
            className="break-words text-sm"
          >
            {error}
          </p>
        </details>
      </div>
    </NoticeBanner>
  )
}
