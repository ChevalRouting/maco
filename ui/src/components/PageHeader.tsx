import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { useSectionTitle } from '@/lib/sectionTitle'
import { setPageHeader, setShellFlag, useShell } from '@/lib/shell'
import { useMediaQuery } from '@/lib/motion'

export interface PageBack {
  to: string
  label: string
}

interface PageHeaderProps {
  title: string
  description?: string
  action?: React.ReactNode
  back?: PageBack
}

export function PageHeader({ title, description, action, back }: PageHeaderProps) {
  const section = useSectionTitle()
  const { toolbar, root, scroller, navActions } = useShell()
  const desktop = useMediaQuery('(min-width: 768px)')
  const heading = useRef<HTMLHeadingElement>(null)
  const phoneTitle = section ?? title
  const backLabel = section ? title : back?.label ?? null

  useEffect(() => {
    setPageHeader({ title: phoneTitle, backLabel })
  }, [phoneTitle, backLabel])

  useEffect(() => () => setPageHeader(null), [])

  useEffect(() => {
    const node = heading.current
    if (desktop || !node || !root) {
      return
    }
    const observer = new IntersectionObserver(
      ([entry]) => setShellFlag(root, 'collapsed', !entry.isIntersecting),
      { root: scroller, rootMargin: '-4px 0px 0px 0px' },
    )
    observer.observe(node)
    return () => {
      observer.disconnect()
      setShellFlag(root, 'collapsed', false)
    }
  }, [desktop, root, scroller])

  if (desktop && toolbar) {
    return createPortal(
      <>
        {back && (
          <Link
            to={back.to}
            aria-label={`Back to ${back.label}`}
            title={back.label}
            className="-ml-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-foreground/70 transition-colors hover:bg-foreground/[0.07] hover:text-foreground active:bg-foreground/[0.12] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <ChevronLeft
              strokeWidth={2.25}
              className="h-5 w-5"
            />
          </Link>
        )}
        <div
          className="min-w-0 flex-1"
        >
          <h1
            title={title}
            className="truncate text-[15px] font-semibold leading-5 tracking-[-0.01em] text-foreground"
          >
            {title}
          </h1>
          {description && (
            <p
              title={description}
              className="truncate text-xs text-muted-foreground"
            >
              {description}
            </p>
          )}
        </div>
        {action && (
          <div
            className="flex shrink-0 items-center gap-2"
          >
            {action}
          </div>
        )}
      </>,
      toolbar,
    )
  }

  const inNavBar = !desktop && navActions !== null

  return (
    <div
      className="flex flex-wrap items-end justify-between gap-x-3 gap-y-3"
    >
      {inNavBar && action && createPortal(action, navActions)}
      <div
        className="min-w-0 flex-1 basis-40"
      >
        <h1
          ref={heading}
          title={phoneTitle}
          className="break-words text-[32px] font-bold leading-[38px] tracking-[-0.025em] text-foreground md:text-[26px] md:leading-tight"
        >
          {desktop ? title : phoneTitle}
        </h1>
        {description && (
          <p
            title={description}
            className="mt-0.5 truncate text-sm text-muted-foreground"
          >
            {description}
          </p>
        )}
      </div>
      {action && !inNavBar && (
        <div
          className="max-w-full shrink-0 pb-0.5"
        >
          {action}
        </div>
      )}
    </div>
  )
}
