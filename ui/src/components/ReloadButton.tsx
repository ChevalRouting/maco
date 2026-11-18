import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/Spinner'

interface ReloadButtonProps {
  onClick: () => void
  loading?: boolean
  label?: string
}

export function ReloadButton({ onClick, loading, label = 'Reload' }: ReloadButtonProps) {
  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={onClick}
      disabled={loading}
      aria-label={label}
      title={label}
    >
      {loading ? (
        <Spinner
          size="sm"
        />
      ) : (
        <RefreshCw
          className="h-4 w-4"
        />
      )}
    </Button>
  )
}
