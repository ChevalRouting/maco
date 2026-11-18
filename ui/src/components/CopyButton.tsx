import { Check, Copy } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useClipboard } from '@/lib/useClipboard'

export function CopyButton({
  text,
  className,
  label = 'Copy to Clipboard',
}: {
  text: string
  className?: string
  label?: string
}) {
  const { copied, copy } = useClipboard()
  return (
    <button
      type="button"
      onClick={() => copy(text)}
      className={cn(
        'inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-muted-foreground hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        className,
      )}
      title={copied ? 'Copied' : label}
      aria-label={copied ? 'Copied' : label}
    >
      <span
        role="status"
        className="sr-only"
      >
        {copied ? 'Copied to clipboard' : ''}
      </span>
      {copied ? (
        <Check
          className="h-3.5 w-3.5 text-success"
        />
      ) : (
        <Copy
          className="h-3.5 w-3.5"
        />
      )}
    </button>
  )
}
