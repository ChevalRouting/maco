import type { ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

export type NoticeIntent = 'info' | 'warning' | 'danger' | 'success'

const intentClasses: Record<NoticeIntent, string> = {
  info: 'bg-info/[0.08] text-foreground [box-shadow:inset_0_0_0_0.5px_hsl(var(--info)/0.3)] [&>span:first-child]:text-info',
  warning: 'bg-warning/[0.1] text-foreground [box-shadow:inset_0_0_0_0.5px_hsl(var(--warning)/0.35)] [&>span:first-child]:text-warning',
  danger: 'bg-danger/[0.08] text-foreground [box-shadow:inset_0_0_0_0.5px_hsl(var(--danger)/0.3)] [&>span:first-child]:text-danger',
  success: 'bg-success/[0.08] text-foreground [box-shadow:inset_0_0_0_0.5px_hsl(var(--success)/0.3)] [&>span:first-child]:text-success',
}

export interface NoticeBannerProps {
  children: ReactNode
  intent?: NoticeIntent
  icon?: ReactNode
  onDismiss?: () => void
  dismissLabel?: string
  className?: string
}

export function NoticeBanner({
  children,
  intent = 'info',
  icon,
  onDismiss,
  dismissLabel = 'Dismiss',
  className,
}: NoticeBannerProps) {
  return (
    <div
      role={intent === 'danger' ? 'alert' : 'status'}
      className={cn('flex min-h-9 items-center gap-2.5 rounded-lg px-3 py-1.5 text-sm', intentClasses[intent], className)}
    >
      {icon && (
        <span
          className="shrink-0"
        >
          {icon}
        </span>
      )}
      <div
        className="min-w-0 flex-1 break-words"
      >
        {children}
      </div>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className="-mr-1.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-muted-foreground hover:bg-foreground/[0.07] hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={dismissLabel}
        >
          <X
            className="h-3.5 w-3.5"
          />
        </button>
      )}
    </div>
  )
}
