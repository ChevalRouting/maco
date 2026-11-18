import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { setSectionTitle } from '@/lib/sectionTitle'

interface SectionItem<T extends string> {
  key: T
  label: string
  title?: string
  badge?: string | number
  group?: string
}

interface SectionNavProps<T extends string> {
  items: SectionItem<T>[]
  active: T
  onChange: (key: T) => void
  children: React.ReactNode
  className?: string
  initiallyEntered?: boolean
}

export function SectionNav<T extends string>({
  items,
  active,
  onChange,
  children,
  className,
  initiallyEntered = false,
}: SectionNavProps<T>) {
  const location = useLocation()
  const navigate = useNavigate()
  const [enteringFrom, setEnteringFrom] = useState<string | null>(null)
  const state = location.state as { sectionEntered?: boolean } | null
  const entered = initiallyEntered || state?.sectionEntered === true
  const activeItem = items.find((i) => i.key === active)
  const activeLabel = activeItem?.title ?? activeItem?.label

  useEffect(() => {
    if (!entered || !activeLabel) {
      return
    }
    setSectionTitle(activeLabel)
    return () => setSectionTitle(null)
  }, [entered, activeLabel])

  useEffect(() => {
    if (enteringFrom === null) {
      return
    }
    const enter = () => {
      setEnteringFrom(null)
      navigate(
        { pathname: location.pathname, search: location.search, hash: location.hash },
        { state: { ...state, sectionEntered: true } },
      )
    }
    if (enteringFrom !== location.key) {
      enter()
      return
    }
    const timer = setTimeout(enter, 100)
    return () => clearTimeout(timer)
  }, [enteringFrom, location, state, navigate])


  const groups: { key: string; title?: string; items: SectionItem<T>[] }[] = []
  for (const item of items) {
    const last = groups[groups.length - 1]
    if (last && last.key === (item.group ?? '')) {
      last.items.push(item)
    } else {
      groups.push({ key: item.group ?? '', items: [item] })
    }
  }
  for (const group of groups) {
    if (group.key && group.items.length > 1) {
      group.title = group.key
    }
  }

  return (
    <div
      className={cn('flex flex-col sm:flex-row sm:gap-6', className)}
    >
      <nav
        className={cn(
          'flex flex-col gap-6 sm:sticky sm:top-3 sm:w-48 sm:shrink-0 sm:gap-4 sm:self-start sm:border-r sm:border-border/70 sm:pr-3',
          entered && 'hidden sm:flex',
        )}
      >
        {groups.map((group) => (
          <div
            key={group.key}
            className="space-y-2 sm:space-y-0.5"
          >
            {group.title && (
              <p
                className="px-1 text-sm font-semibold text-foreground sm:px-2.5 sm:pb-0.5 sm:text-xs sm:text-muted-foreground/80"
              >
                {group.title}
              </p>
            )}
            <div
              className="inset-separators flex flex-col overflow-hidden rounded-xl bg-card [box-shadow:var(--card-shadow)] sm:gap-px sm:overflow-visible sm:rounded-none sm:bg-transparent sm:[box-shadow:none] sm:[&>*::before]:hidden"
            >
              {group.items.map((it) => (
                <button
                  key={it.key}
                  type="button"
                  onClick={() => {
                    if (!entered && window.matchMedia('(max-width: 639px)').matches) {
                      setEnteringFrom(it.key === active ? '' : location.key)
                    }
                    onChange(it.key)
                  }}
                  className={cn(
                    'flex min-h-[2.75rem] items-center gap-2 px-4 text-left text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:bg-foreground/[0.06] sm:min-h-0 sm:rounded-md sm:px-2.5 sm:py-[5px]',
                    active === it.key
                      ? 'sm:bg-foreground/[0.09] sm:font-medium sm:active:bg-foreground/[0.09] dark:sm:bg-white/[0.12]'
                      : 'sm:active:bg-foreground/[0.04]',
                  )}
                >
                  <span
                    className="min-w-0 flex-1 truncate"
                  >
                    {it.label}
                  </span>
                  {it.badge != null && (
                    <span
                      className="shrink-0 text-xs tabular-nums text-muted-foreground"
                    >
                      {it.badge}
                    </span>
                  )}
                  <ChevronRight
                    strokeWidth={2.5}
                    className="h-3.5 w-3.5 shrink-0 text-muted-foreground/50 sm:hidden"
                  />
                </button>
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div
        className={cn('min-w-0 flex-1 pt-4 sm:pt-0', !entered && 'hidden sm:block')}
      >
        {children}
      </div>
    </div>
  )
}
