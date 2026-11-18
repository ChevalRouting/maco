import { Button } from 'maco-ui'
import { Plus } from 'lucide-react'
import { useCompact } from '../hooks/useNarrow'

interface CreateButtonProps {
  label: string
  onClick: () => void
  toolbar?: boolean
}

export function CreateButton({ label, onClick, toolbar = false }: CreateButtonProps) {
  const compact = useCompact()

  if (toolbar && compact) {
    return (
      <Button variant="ghost" size="icon" aria-label={label} title={label} onClick={onClick}>
        <Plus aria-hidden="true" strokeWidth={2.25} className="h-6 w-6 text-primary" />
      </Button>
    )
  }

  return (
    <Button variant="suggested" className="gap-2" onClick={onClick}>
      <Plus aria-hidden="true" className="h-4 w-4" />
      {label}
    </Button>
  )
}
