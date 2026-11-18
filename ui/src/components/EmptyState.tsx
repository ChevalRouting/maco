import * as React from 'react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon?: React.ReactNode
  title?: string
  message?: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({ icon, title, message, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-2.5 rounded-xl px-6 py-16 text-center',
        className,
      )}
    >
      {icon && (
        <div
          className="mb-1 text-muted-foreground/35 [&_svg]:h-14 [&_svg]:w-14 [&_svg]:stroke-[1.25]"
        >
          {icon}
        </div>
      )}
      {title && (
        <p
          className="text-base font-semibold text-foreground"
        >
          {title}
        </p>
      )}
      {message && (
        <p
          className="max-w-sm text-sm text-muted-foreground"
        >
          {message}
        </p>
      )}
      {action}
    </div>
  )
}
