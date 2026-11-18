import { useEffect, useRef, useState } from 'react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'
import * as Dialog from '@radix-ui/react-dialog'
import { Check, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { NavItem } from '@/layouts/AppShell'

export interface MobileTab {
  key: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  to?: string
  exact?: boolean
  items?: NavItem[]
  sheet?: React.ReactNode
}

function pathMatches(pathname: string, to: string, exact?: boolean) {
  return exact ? pathname === to : pathname === to || pathname.startsWith(`${to}/`)
}

function tabMatches(tab: MobileTab, pathname: string) {
  if (tab.to && pathMatches(pathname, tab.to, tab.exact)) {
    return true
  }
  return (tab.items ?? []).some((item) => pathMatches(pathname, item.to, item.exact))
}

export function MobileSubNav({ items, className }: { items: NavItem[]; className?: string }) {
  const { pathname } = useLocation()
  const scroller = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const active = scroller.current?.querySelector<HTMLElement>('[aria-current="page"]')
    active?.scrollIntoView({ inline: 'center', block: 'nearest' })
  }, [pathname])

  return (
    <div
      ref={scroller}
      className={cn(
        'flex items-center gap-1.5 overflow-x-auto scrollbar-none px-3 py-2',
        className,
      )}
    >
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.exact}
          className={({ isActive }) =>
            cn(
              'flex items-center gap-1.5 shrink-0 rounded-full px-3 py-1.5 text-xs font-medium transition-colors',
              isActive
                ? 'bg-card text-foreground [box-shadow:var(--control-shadow)] dark:bg-secondary'
                : 'text-sidebar-foreground/75 hover:text-sidebar-foreground',
            )
          }
        >
          <item.icon
            className="h-3.5 w-3.5 shrink-0 opacity-80"
          />
          {item.label}
        </NavLink>
      ))}
    </div>
  )
}

function MoreSheet({ tab, onClose }: { tab: MobileTab; onClose: () => void }) {
  const items = tab.items ?? []
  return (
    <Dialog.Root
      open
      onOpenChange={(o) => {
        if (!o) {
          onClose()
        }
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay
          className="fixed inset-0 z-40 bg-black/25 md:hidden data-[state=open]:animate-in data-[state=open]:fade-in-0"
        />
        <Dialog.Content
          aria-describedby={undefined}
          onOpenAutoFocus={(e) => e.preventDefault()}
          className="fixed inset-x-0 bottom-0 z-50 flex max-h-[75vh] flex-col rounded-t-[14px] bg-background [box-shadow:var(--overlay-shadow)] focus:outline-none md:hidden data-[state=open]:animate-in data-[state=open]:slide-in-from-bottom-8 data-[state=open]:duration-300"
          onEscapeKeyDown={onClose}
          onInteractOutside={onClose}
        >
          <div
            className="flex shrink-0 justify-center pb-0.5 pt-1.5"
          >
            <div
              className="h-[5px] w-9 rounded-full bg-muted-foreground/30"
            />
          </div>

          <div
            className="grid h-11 shrink-0 grid-cols-[4rem_minmax(0,1fr)_4rem] items-center px-4"
          >
            <span
              aria-hidden="true"
            />
            <Dialog.Title
              className="truncate text-center text-sm font-semibold"
            >
              {tab.label}
            </Dialog.Title>
            <Dialog.Close
              asChild
            >
              <button
                type="button"
                className="justify-self-end text-sm font-semibold text-primary active:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded"
              >
                Done
              </button>
            </Dialog.Close>
          </div>

          <div
            className="space-y-6 overflow-y-auto px-4 pb-6 pt-2 safe-bottom"
          >
            {items.length > 0 && (
              <div
                className="inset-separators overflow-hidden rounded-xl bg-card [box-shadow:var(--card-shadow)] [&>*+*]:before:!left-12"
              >
                {items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.exact}
                    onClick={onClose}
                    className="flex min-h-[2.75rem] items-center gap-3 px-4 text-sm text-foreground active:bg-foreground/[0.06]"
                  >
                    {({ isActive }) => (
                      <>
                        <item.icon
                          className="h-5 w-5 shrink-0 text-primary"
                        />
                        <span
                          className="min-w-0 flex-1 truncate"
                        >
                          {item.label}
                        </span>
                        {isActive ? (
                          <Check
                            strokeWidth={2.5}
                            className="h-4 w-4 shrink-0 text-primary"
                          />
                        ) : (
                          <ChevronRight
                            strokeWidth={2.5}
                            className="h-3.5 w-3.5 shrink-0 text-muted-foreground/50"
                          />
                        )}
                      </>
                    )}
                  </NavLink>
                ))}
              </div>
            )}
            {tab.sheet}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

interface MobileNavProps {
  tabs: MobileTab[]
  activeKey?: string
  className?: string
  subNav?: boolean
}

export function MobileNav({ tabs, activeKey, className, subNav = true }: MobileNavProps) {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const [sheetKey, setSheetKey] = useState<string | null>(null)

  useEffect(() => setSheetKey(null), [pathname])

  const activeTab = activeKey
    ? tabs.find((tab) => tab.key === activeKey)
    : tabs.find((tab) => tabMatches(tab, pathname))
  const sheetTab = sheetKey ? tabs.find((tab) => tab.key === sheetKey) : undefined

  const handleTap = (tab: MobileTab) => {
    if (tab.to) {
      navigate(tab.to)
      return
    }
    if (tab.sheet) {
      setSheetKey(tab.key)
      return
    }
    if (tab.items?.length && tab.key !== activeTab?.key) {
      navigate(tab.items[0].to)
    }
  }

  return (
    <nav
      className={cn(
        'material md:hidden shrink-0 safe-bottom border-t border-foreground/10',
        className,
      )}
    >
      {subNav && activeTab?.items && activeTab.items.length > 0 && (
        <div
          className="border-b border-foreground/10"
        >
          <MobileSubNav
            items={activeTab.items}
          />
        </div>
      )}

      <div
        className="h-14 flex items-stretch"
      >
        {tabs.map((tab) => {
          const isActive = tab.key === activeTab?.key
          return (
            <button
              key={tab.key}
              type="button"
              onClick={() => handleTap(tab)}
              className={cn(
                'relative flex-1 flex flex-col items-center justify-center gap-0.5 transition-colors',
                isActive
                  ? 'text-sidebar-primary'
                  : 'text-muted-foreground hover:text-sidebar-foreground',
              )}
            >
              <tab.icon
                className="h-5 w-5"
              />
              <span
                className="text-[10px] font-medium leading-none"
              >
                {tab.label}
              </span>
            </button>
          )
        })}
      </div>

      {sheetTab && (
        <MoreSheet
          tab={sheetTab}
          onClose={() => setSheetKey(null)}
        />
      )}
    </nav>
  )
}
