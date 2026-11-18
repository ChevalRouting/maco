import { Suspense, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigationType } from 'react-router-dom'
import { Moon, MoreHorizontal, PanelLeft, Sun } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useTheme } from '@/lib/useTheme'
import { SectionLabel } from '@/components/SectionLabel'
import { Spinner } from '@/components/Spinner'
import { MobileNav, type MobileTab } from '@/components/ui/mobile-nav'
import { COMPACT_QUERY, EASE_OUT, EASE_SHEET, matches, reducedMotion, useMediaQuery } from '@/lib/motion'
import { ShellContext, setShellFlag } from '@/lib/shell'

export interface NavItem {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  exact?: boolean
}

export interface NavGroup {
  label?: string
  key?: string
  icon?: React.ComponentType<{ className?: string }>
  items: NavItem[]
}

const MAX_MOBILE_TABS = 5

function buildMobileTabs(nav: NavGroup[], extras: React.ReactNode): MobileTab[] {
  if (nav.length > 1) {
    const tabs = nav.map((group, gi): MobileTab => {
      const direct = group.items.length === 1 && !group.label
      const icon = group.icon ?? group.items[0]?.icon ?? MoreHorizontal
      if (direct) {
        const item = group.items[0]
        return {
          key: group.key ?? item.to,
          label: group.label || item.label,
          icon,
          to: item.to,
          exact: item.exact,
        }
      }
      return {
        key: group.key ?? group.label ?? `group-${gi}`,
        label: group.label ?? 'Menu',
        icon,
        items: group.items,
      }
    })
    if (extras) {
      tabs.push({ key: '__more__', label: 'More', icon: MoreHorizontal, sheet: extras })
    }
    return tabs
  }

  const items = nav[0]?.items ?? []
  const primary = items.slice(0, MAX_MOBILE_TABS - 1)
  const overflow = items.slice(MAX_MOBILE_TABS - 1)
  const tabs: MobileTab[] = primary.map((item) => ({
    key: item.to,
    label: item.label,
    icon: item.icon,
    to: item.to,
    exact: item.exact,
  }))
  tabs.push({
    key: '__more__',
    label: 'More',
    icon: MoreHorizontal,
    items: overflow,
    sheet: extras,
  })
  return tabs
}

interface ShellSlotState {
  collapsed: boolean
  closeMobile: () => void
}

type ShellSlot = React.ReactNode | ((state: ShellSlotState) => React.ReactNode)

interface AppShellProps {
  nav: NavGroup[]
  title?: string
  logo?: React.ComponentType<{ className?: string }>
  brand?: React.ReactNode
  banners?: React.ReactNode
  sidebarTop?: ShellSlot
  sidebarFooter?: ShellSlot
  mobileNavigation?: React.ReactNode
  mobileOverlay?: React.ReactNode
  showMobileHeader?: boolean
  showThemeToggle?: boolean
  collapsedStorageKey?: string
  contentClassName?: string
  isRootPath?: (pathname: string) => boolean
  children?: React.ReactNode
}

const SIDEBAR_MOTION = 300

function renderSlot(slot: ShellSlot | undefined, state: ShellSlotState) {
  return typeof slot === 'function' ? slot(state) : slot
}

function NavItems({
  groups,
  onNavigate,
}: {
  groups: NavGroup[]
  onNavigate?: () => void
}) {
  return (
    <nav
      className="space-y-4 px-2.5"
    >
      {groups.map((group, gi) => (
        <div
          key={group.label || `group-${gi}`}
        >
          {group.label && (
            <div
              className="mb-1 px-2"
            >
              <SectionLabel
                className="text-xs font-semibold text-sidebar-foreground/45"
              >
                {group.label}
              </SectionLabel>
            </div>
          )}
          <div
            className="space-y-px"
          >
            {group.items.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.exact}
                onClick={onNavigate}
                aria-label={item.label}
                className={({ isActive }) =>
                  cn(
                    'group flex h-7 w-full items-center gap-2 rounded-md px-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                    isActive
                      ? 'bg-foreground/[0.09] font-medium text-sidebar-foreground dark:bg-white/[0.12]'
                      : 'text-sidebar-foreground/90 active:bg-foreground/[0.05]',
                  )
                }
              >
                <item.icon
                  className="h-4 w-4 shrink-0 text-primary"
                />
                <span
                  className="truncate"
                >
                  {item.label}
                </span>
              </NavLink>
            ))}
          </div>
        </div>
      ))}
    </nav>
  )
}

function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const icon = theme === 'dark' ? (
    <Sun
      className="h-3.5 w-3.5"
    />
  ) : (
    <Moon
      className="h-3.5 w-3.5"
    />
  )
  const label = theme === 'dark' ? 'Light mode' : 'Dark mode'

  return (
    <button
      onClick={toggle}
      title={label}
      aria-label={label}
      className="flex w-full items-center gap-2.5 rounded-md px-3 py-1.5 text-sm font-medium text-sidebar-foreground/80 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
    >
      {icon}
      {label}
    </button>
  )
}

function Brand({
  title,
  brand,
  logo,
}: {
  title: string
  brand?: React.ReactNode
  logo?: React.ComponentType<{ className?: string }>
}) {
  if (brand) {
    return (
      <>
        {brand}
      </>
    )
  }

  const Logo = logo

  return (
    <>
      {Logo && (
        <Logo
          className="h-7 w-7 shrink-0"
        />
      )}
      <span
        className="truncate text-[15px] font-semibold tracking-tight text-sidebar-foreground"
      >
        {title}
      </span>
    </>
  )
}

function SidebarToggle({ visible, onToggle }: { visible: boolean; onToggle: () => void }) {
  const label = visible ? 'Hide Sidebar' : 'Show Sidebar'
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-label={label}
      title={`${label} (Control-Command-S)`}
      aria-expanded={visible}
      className="flex h-7 w-8 shrink-0 items-center justify-center rounded-md text-foreground/60 transition-colors hover:bg-foreground/[0.07] hover:text-foreground active:bg-foreground/[0.12] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      <PanelLeft
        className="h-4 w-4"
      />
    </button>
  )
}

interface NavState {
  sectionEntered?: boolean
}

function useMobileTransition(
  content: React.RefObject<HTMLDivElement>,
  scroller: React.RefObject<HTMLElement>,
  isRootPath?: (pathname: string) => boolean,
) {
  const location = useLocation()
  const navigationType = useNavigationType()
  const entered = (location.state as NavState | null)?.sectionEntered === true
  const depth = (isRootPath ? (isRootPath(location.pathname) ? 0 : 1) : 0) + (entered ? 1 : 0)
  const previous = useRef({ depth, pathname: location.pathname, key: location.key })
  const offsets = useRef(new Map<string, number>())

  useLayoutEffect(() => {
    const last = previous.current
    previous.current = { depth, pathname: location.pathname, key: location.key }
    if (last.key === location.key) {
      return
    }
    const main = scroller.current
    if (main) {
      offsets.current.set(last.key, main.scrollTop)
      if (last.pathname !== location.pathname || last.depth !== depth) {
        main.scrollTop = navigationType === 'POP' ? offsets.current.get(location.key) ?? 0 : 0
      }
    }
    const node = content.current
    if (!node || depth === last.depth || !matches(COMPACT_QUERY) || reducedMotion()) {
      return
    }
    const forward = depth > last.depth
    node.animate(
      [
        { transform: `translateX(${forward ? 32 : -24}%)`, opacity: 0 },
        { opacity: 1, offset: 0.6 },
        { transform: 'translateX(0)', opacity: 1 },
      ],
      { duration: 380, easing: EASE_SHEET },
    )
  }, [location.key, location.pathname, depth, navigationType, content, scroller])
}

export function AppShell({
  nav,
  title = 'App',
  logo,
  brand,
  banners,
  sidebarTop,
  sidebarFooter,
  mobileNavigation,
  mobileOverlay,
  showMobileHeader = true,
  showThemeToggle = true,
  collapsedStorageKey = 'sidebar-collapsed',
  contentClassName,
  isRootPath,
  children,
}: AppShellProps) {
  const [docked, setDocked] = useState(() => {
    try {
      return localStorage.getItem(collapsedStorageKey) !== 'true'
    } catch {
      return true
    }
  })
  const wide = useMediaQuery('(min-width: 1024px)')
  const [overlayOpen, setOverlayOpen] = useState(false)
  const visible = wide ? docked : overlayOpen
  const pushed = wide && docked
  const [toolbar, setToolbar] = useState<HTMLElement | null>(null)
  const [root, setRoot] = useState<HTMLElement | null>(null)
  const [scroller, setScroller] = useState<HTMLElement | null>(null)
  const [navActions, setNavActions] = useState<HTMLElement | null>(null)
  const pane = useRef<HTMLDivElement>(null)
  const aside = useRef<HTMLElement>(null)
  const content = useRef<HTMLDivElement>(null)
  const scrollerRef = useRef<HTMLElement | null>(null)
  const sentinel = useRef<HTMLDivElement>(null)
  const paneLeft = useRef<number | null>(null)
  const closeMobile = () => {}

  const bindScroller = useCallback((node: HTMLElement | null) => {
    scrollerRef.current = node
    setScroller(node)
  }, [])

  useMobileTransition(content, scrollerRef, isRootPath)

  const location = useLocation()

  useEffect(() => setOverlayOpen(false), [location.pathname, wide])

  const toggleSidebar = useCallback(() => {
    if (!wide) {
      setOverlayOpen((value) => !value)
      return
    }
    paneLeft.current = pane.current?.getBoundingClientRect().left ?? null
    setDocked((value) => {
      try {
        localStorage.setItem(collapsedStorageKey, String(value))
      } catch {
        return !value
      }
      return !value
    })
  }, [collapsedStorageKey, wide])

  useLayoutEffect(() => {
    aside.current?.toggleAttribute('inert', !visible)
    const before = paneLeft.current
    const node = pane.current
    paneLeft.current = null
    if (before === null || !node || reducedMotion()) {
      return
    }
    const delta = before - node.getBoundingClientRect().left
    if (delta !== 0) {
      node.animate(
        [{ transform: `translateX(${delta}px)` }, { transform: 'translateX(0)' }],
        { duration: SIDEBAR_MOTION, easing: EASE_OUT },
      )
    }
  }, [visible, pushed])

  useEffect(() => {
    const handleKey = (event: KeyboardEvent) => {
      if (event.ctrlKey && event.metaKey && event.key.toLowerCase() === 's') {
        event.preventDefault()
        toggleSidebar()
      }
      if (event.key === 'Escape' && overlayOpen && !wide) {
        setOverlayOpen(false)
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [toggleSidebar, overlayOpen, wide])

  useEffect(() => {
    const node = sentinel.current
    if (!node || !scroller || !root) {
      return
    }
    const observer = new IntersectionObserver(
      ([entry]) => setShellFlag(root, 'scrolled', !entry.isIntersecting),
      { root: scroller },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [scroller, root])

  const shell = useMemo(
    () => ({ toolbar, root, scroller, navActions, setNavActions }),
    [toolbar, root, scroller, navActions],
  )

  const mobileExtras = (
    <div
      className="mt-2 pt-2 border-t border-border/50 space-y-0.5"
    >
      {sidebarTop && (
        <div
          className="pb-1"
        >
          {renderSlot(sidebarTop, { collapsed: false, closeMobile })}
        </div>
      )}
      {renderSlot(sidebarFooter, { collapsed: false, closeMobile })}
      {showThemeToggle && <ThemeToggle />}
    </div>
  )

  return (
    <ShellContext.Provider
      value={shell}
    >
      <div
        ref={setRoot}
        className="app-shell relative flex h-dvh overflow-hidden bg-background"
      >
        {!wide && overlayOpen && (
          <div
            aria-hidden="true"
            className="absolute inset-0 z-[15] hidden md:block"
            onPointerDown={() => setOverlayOpen(false)}
          />
        )}
        <aside
          ref={aside}
          aria-hidden={visible ? undefined : true}
          className={cn(
            'absolute bottom-2 left-2 top-2 z-20 hidden w-60 flex-col overflow-hidden rounded-2xl bg-sidebar text-sidebar-foreground md:flex',
            pushed ? '[box-shadow:var(--card-shadow)]' : '[box-shadow:var(--overlay-shadow)]',
            'transition-[transform,visibility] duration-300 ease-[cubic-bezier(0.2,0.9,0.25,1)] motion-reduce:transition-none',
            visible ? 'translate-x-0' : 'invisible -translate-x-[calc(100%+1rem)]',
          )}
        >
          <div
            className="flex h-14 shrink-0 items-center gap-2.5 pl-3.5 pr-2"
          >
            <Brand
              title={title}
              brand={brand}
              logo={logo}
            />
            <span
              className="flex-1"
            />
            <SidebarToggle
              visible
              onToggle={toggleSidebar}
            />
          </div>

          {sidebarTop && (
            <div
              className="px-2 pb-2"
            >
              {renderSlot(sidebarTop, { collapsed: false, closeMobile })}
            </div>
          )}

          <div
            className="flex-1 overflow-y-auto overflow-x-hidden py-2"
          >
            <NavItems
              groups={nav}
            />
          </div>

          {(sidebarFooter || showThemeToggle) && (
            <div
              className="space-y-0.5 p-2"
            >
              {renderSlot(sidebarFooter, { collapsed: false, closeMobile })}
              {showThemeToggle && <ThemeToggle />}
            </div>
          )}
        </aside>

        <div
          ref={pane}
          className={cn(
            'flex min-w-0 flex-1 flex-col overflow-hidden bg-background',
            pushed ? 'md:ml-[16rem]' : 'md:ml-0',
          )}
        >
          {showMobileHeader && (
            <div
              className="material md:hidden flex items-center gap-2 h-12 px-3 border-b border-border/70 text-sidebar-foreground shrink-0"
            >
              <Brand
                title={title}
                brand={brand}
                logo={logo}
              />
            </div>
          )}

          {banners}

          <header
            className={cn(
              'window-toolbar mt-2 hidden h-14 shrink-0 items-center gap-2 pr-4 md:flex lg:pr-6',
              pushed ? 'pl-4 lg:pl-6' : 'pl-4',
            )}
          >
            {!pushed && (
              <SidebarToggle
                visible={false}
                onToggle={toggleSidebar}
              />
            )}
            <div
              ref={setToolbar}
              className="flex min-w-0 flex-1 items-center gap-3"
            />
          </header>

          <main
            ref={bindScroller}
            className="flex-1 overflow-auto overscroll-contain"
          >
            <div
              ref={sentinel}
              aria-hidden="true"
              className="h-px"
            />
            <div
              ref={content}
              className={cn('px-4 pb-6 pt-1 md:px-4 md:pb-7 md:pt-3 lg:px-6', contentClassName)}
            >
              <Suspense
                fallback={<Spinner />}
              >
                {children ?? <Outlet />}
              </Suspense>
            </div>
          </main>
          {mobileNavigation ?? (
            <MobileNav
              tabs={buildMobileTabs(nav, mobileExtras)}
            />
          )}
        </div>

        {mobileOverlay}
      </div>
    </ShellContext.Provider>
  )
}
