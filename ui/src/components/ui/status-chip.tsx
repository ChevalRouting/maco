import { cn } from '@/lib/utils'

export type StatusIntent = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

interface StatusChipProps {
  intent: StatusIntent
  label: string
  pulse?: boolean
  className?: string
}

const CHIP_STYLES: Record<StatusIntent, string> = {
  success: 'text-foreground',
  warning: 'text-foreground',
  danger: 'text-foreground',
  info: 'text-foreground',
  neutral: 'text-muted-foreground',
}

const DOT_STYLES: Record<StatusIntent, string> = {
  success: 'bg-success',
  warning: 'bg-warning',
  danger: 'bg-danger',
  info: 'bg-info',
  neutral: 'bg-muted-foreground/50',
}

export function StatusChip({ intent, label, pulse = false, className }: StatusChipProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 whitespace-nowrap text-sm',
        CHIP_STYLES[intent],
        className,
      )}
    >
      <span
        className={cn(
          'h-2 w-2 rounded-full shrink-0 [box-shadow:inset_0_0_0_0.5px_hsl(0_0%_0%/0.12)]',
          DOT_STYLES[intent],
          pulse && 'animate-pulse',
        )}
      />
      {label}
    </span>
  )
}
