import { type LucideIcon } from 'lucide-react'
import { Meter } from '@/components/Meter'
import { Card } from '@/components/ui/card'

export interface StatCardProps {
  icon: LucideIcon
  label: string
  value: string
  caption?: string
  pct?: number
  meta?: string
  warn?: boolean
  barLabel?: string
}

export function StatCard({
  icon: Icon,
  label,
  value,
  caption,
  pct,
  meta,
  warn,
  barLabel,
}: StatCardProps) {
  return (
    <Card
      className="space-y-3 p-4"
    >
      <div
        className="flex items-center gap-2 text-muted-foreground"
      >
        <Icon
          aria-hidden="true"
          className="h-4 w-4 shrink-0"
        />
        <span
          className="text-sm font-semibold"
        >
          {label}
        </span>
      </div>

      <div>
        <p
          className="text-2xl font-semibold leading-tight tabular-nums"
        >
          {value}
        </p>
        {caption && (
          <p
            className="mt-1 text-sm text-muted-foreground"
          >
            {caption}
          </p>
        )}
      </div>

      {pct !== undefined && (
        <Meter
          value={pct}
          label={barLabel || label}
          warn={warn}
        />
      )}

      {meta && (
        <p
          className="text-sm text-muted-foreground tabular-nums"
        >
          {meta}
        </p>
      )}
    </Card>
  )
}
