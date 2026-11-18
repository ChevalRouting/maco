import { cn } from '@/lib/utils'

interface TabItem<T extends string> {
  key: T
  label: string
  badge?: string | number
}

interface TabsProps<T extends string> {
  tabs: TabItem<T>[]
  active: T
  onChange: (key: T) => void
  variant?: 'underline' | 'pills'
  className?: string
  id?: string
  label?: string
}

export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
  variant = 'underline',
  className,
  id,
  label,
}: TabsProps<T>) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) {
      return
    }
    event.preventDefault()

    let nextIndex = index
    if (event.key === 'ArrowLeft') {
      nextIndex = (index - 1 + tabs.length) % tabs.length
    }
    if (event.key === 'ArrowRight') {
      nextIndex = (index + 1) % tabs.length
    }
    if (event.key === 'Home') {
      nextIndex = 0
    }
    if (event.key === 'End') {
      nextIndex = tabs.length - 1
    }

    onChange(tabs[nextIndex].key)
    event.currentTarget.parentElement
      ?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
      [nextIndex]?.focus()
    event.currentTarget.parentElement
      ?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
      [nextIndex]?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
  }

  return (
    <div
      role="tablist"
      aria-label={label}
      className={cn(
        'flex max-w-full gap-0.5 overflow-x-auto rounded-lg bg-foreground/[0.06] p-0.5 scrollbar-none',
        variant === 'underline' && 'w-fit self-start',
        className,
      )}
    >
      {tabs.map((t, index) => (
        <button
          key={t.key}
          type="button"
          role="tab"
          id={id ? `${id}-tab-${t.key}` : undefined}
          aria-controls={id ? `${id}-panel-${t.key}` : undefined}
          aria-selected={active === t.key}
          tabIndex={active === t.key ? 0 : -1}
          onClick={() => onChange(t.key)}
          onKeyDown={(event) => handleKeyDown(event, index)}
          className={cn(
            'flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md px-3 py-[3px] text-sm font-medium transition-[background-color,box-shadow] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50',
            active === t.key
              ? 'bg-card text-foreground [box-shadow:var(--control-shadow)] dark:bg-secondary'
              : 'text-foreground/80',
          )}
        >
          {t.label}
          {t.badge != null && (
            <span
              className={cn(
                'tabular-nums text-xs font-normal text-muted-foreground',
              )}
            >
              {t.badge}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}
