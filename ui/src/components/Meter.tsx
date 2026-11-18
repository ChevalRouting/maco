import { cn } from '@/lib/utils'

export interface MeterProps {
  value: number
  label: string
  warn?: boolean
  colorClass?: string
  className?: string
}

export function Meter({ value, label, warn, colorClass, className }: MeterProps) {
  const fill = Number.isFinite(value) ? Math.max(0, Math.min(100, value)) : 0
  return (
    <div
      role="meter"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={fill}
      aria-valuetext={`${Math.round(fill)}%`}
      className={cn('h-1.5 w-full overflow-hidden rounded-full bg-foreground/[0.08] [box-shadow:inset_0_0_0_0.5px_hsl(var(--foreground)/0.06)]', className)}
    >
      <div
        className={cn('h-full rounded-full', warn ? 'bg-warning' : 'bg-primary', colorClass)}
        style={{ width: `${fill}%` }}
      />
    </div>
  )
}
