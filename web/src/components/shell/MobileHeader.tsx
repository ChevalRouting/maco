import { useLocation, useNavigate } from 'react-router-dom'
import { NavBarActions, usePageHeader } from 'maco-ui'
import { ChevronLeft } from 'lucide-react'
import { isTopLevel } from '../../lib/navigation'
import { Logo } from '../../Logo'

interface SectionState {
  sectionEntered?: boolean
}

export function MobileHeader() {
  const navigate = useNavigate()
  const location = useLocation()
  const header = usePageHeader()
  const entered = (location.state as SectionState | null)?.sectionEntered === true
  const root = isTopLevel(location.pathname) && !entered
  const backLabel = header?.backLabel ?? 'Back'

  return (
    <header className="nav-bar safe-top shrink-0 bg-background text-foreground md:hidden">
      <div className="flex h-11 items-center gap-2 pl-2 pr-12">
        {root ? (
          <Logo className="ml-1.5 h-7 w-7 shrink-0" />
        ) : (
          <button
            type="button"
            onClick={() => navigate(-1)}
            aria-label={`Back to ${backLabel}`}
            title={backLabel}
            className="-ml-1 flex h-11 min-w-0 max-w-[40%] shrink items-center gap-0.5 pr-1 text-base text-primary active:opacity-50"
          >
            <ChevronLeft strokeWidth={2.5} className="h-6 w-6 shrink-0" />
            <span className="truncate">{backLabel}</span>
          </button>
        )}
        <span className="nav-bar-title min-w-0 flex-1 truncate text-center text-base font-semibold" aria-hidden="true">
          {header?.title}
        </span>
        <NavBarActions className="flex shrink-0 items-center gap-2 empty:hidden" />
      </div>
    </header>
  )
}
