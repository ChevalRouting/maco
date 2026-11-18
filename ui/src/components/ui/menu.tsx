import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EASE_SHEET, ExitMarker, PHONE_QUERY, fadeOut, matches, useMediaQuery } from '@/lib/motion'

type MenuIcon = React.ComponentType<{ className?: string }>

export interface MenuItem {
  key: string
  label: string
  icon?: MenuIcon
  onSelect: () => void
  checked?: boolean
  destructive?: boolean
  disabled?: boolean
}

export interface MenuButtonProps {
  label: string
  sections: MenuItem[][]
  children: React.ReactNode
  align?: 'start' | 'end'
  className?: string
  menuClassName?: string
}

interface MenuPosition {
  top: number
  left?: number
  right?: number
  above?: boolean
  maxHeight?: number
}

const MENU_GAP = 4

const VIEWPORT_MARGIN = 8

function menuPosition(trigger: HTMLElement, align: 'start' | 'end', height: number): MenuPosition {
  const rect = trigger.getBoundingClientRect()
  const below = window.innerHeight - rect.bottom - MENU_GAP - VIEWPORT_MARGIN
  const above = rect.top - MENU_GAP - VIEWPORT_MARGIN
  const flip = height > below && above > below
  const top = flip
    ? Math.max(VIEWPORT_MARGIN, rect.top - MENU_GAP - height)
    : rect.bottom + MENU_GAP
  const horizontal =
    align === 'start' ? { left: rect.left } : { right: window.innerWidth - rect.right }
  return { top, above: flip, maxHeight: flip ? above : below, ...horizontal }
}

function menuItems(menu: HTMLElement | null): HTMLButtonElement[] {
  return Array.from(
    menu?.querySelectorAll<HTMLButtonElement>('[role^="menuitem"]:not(:disabled)') ?? [],
  )
}

function moveFocus(menu: HTMLElement | null, step: number) {
  const items = menuItems(menu)
  if (items.length === 0) {
    return
  }

  const index = items.indexOf(document.activeElement as HTMLButtonElement)
  items[(index + step + items.length) % items.length].focus()
}

export function MenuButton({
  label,
  sections,
  children,
  align = 'end',
  className,
  menuClassName,
}: MenuButtonProps) {
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState<MenuPosition>({ top: 0 })
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const visibleSections = sections.filter((section) => section.length > 0)

  useLayoutEffect(() => {
    if (open && triggerRef.current) {
      setPosition(menuPosition(triggerRef.current, align, menuRef.current?.offsetHeight ?? 0))
    }
  }, [open, align])

  useEffect(() => {
    if (!open) {
      return
    }

    menuItems(menuRef.current)[0]?.focus()
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node
      if (!menuRef.current?.contains(target) && !triggerRef.current?.contains(target)) {
        setOpen(false)
      }
    }
    const handleClose = () => setOpen(false)
    const sheet = matches(PHONE_QUERY)
    document.addEventListener('pointerdown', handlePointerDown)
    window.addEventListener('resize', handleClose)
    if (!sheet) {
      window.addEventListener('scroll', handleClose, true)
    }
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      window.removeEventListener('resize', handleClose)
      window.removeEventListener('scroll', handleClose, true)
    }
  }, [open])

  const close = () => {
    setOpen(false)
    triggerRef.current?.focus()
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape' || event.key === 'Tab') {
      event.preventDefault()
      close()
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      moveFocus(menuRef.current, 1)
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      moveFocus(menuRef.current, -1)
    }
  }

  const handleSelect = (item: MenuItem) => {
    close()
    item.onSelect()
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        aria-label={label}
        title={label}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className={cn(
          'inline-flex items-center justify-center gap-2 rounded-lg text-sm font-medium text-muted-foreground transition-colors hover:bg-foreground/[0.06] hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
          open && 'bg-foreground/[0.08] text-foreground',
          className,
        )}
      >
        {children}
      </button>
      {open && (
        <MenuPanel
          panelRef={menuRef}
          label={label}
          sections={visibleSections}
          position={position}
          origin={`${position.above ? 'bottom' : 'top'} ${align === 'start' ? 'left' : 'right'}`}
          onKeyDown={handleKeyDown}
          onSelect={handleSelect}
          onDismiss={close}
          className={menuClassName}
        />
      )}
    </>
  )
}

interface MenuPanelProps {
  panelRef: React.RefObject<HTMLDivElement>
  label: string
  sections: MenuItem[][]
  position: MenuPosition
  origin: string
  onKeyDown: (event: React.KeyboardEvent<HTMLDivElement>) => void
  onSelect: (item: MenuItem) => void
  onDismiss: () => void
  className?: string
}

function sheetExit(): Keyframe[] {
  return [{ transform: 'translateY(0)' }, { transform: 'translateY(110%)' }]
}

function MenuPanel({
  panelRef,
  label,
  sections,
  position,
  origin,
  onKeyDown,
  onSelect,
  onDismiss,
  className,
}: MenuPanelProps) {
  const phone = useMediaQuery(PHONE_QUERY)

  if (phone) {
    return createPortal(
      <div
        className="fixed inset-0 z-50"
      >
        <div
          aria-hidden="true"
          className="motion-fade-in absolute inset-0 bg-black/25"
        >
          <ExitMarker
            keyframes={fadeOut}
            duration={240}
          />
        </div>
        <div
          ref={panelRef}
          role="menu"
          aria-label={label}
          onKeyDown={onKeyDown}
          className="motion-rise-in absolute inset-x-2 bottom-[calc(env(safe-area-inset-bottom,0px)+0.5rem)] space-y-2"
        >
          <ExitMarker
            keyframes={sheetExit}
            duration={280}
            easing={EASE_SHEET}
          />
          <div
            className="overflow-hidden rounded-[14px] bg-popover text-popover-foreground"
          >
            <p
              className="truncate border-b border-foreground/10 px-4 py-3 text-center text-xs font-medium text-muted-foreground"
            >
              {label}
            </p>
            {sections.map((section, index) => (
              <div
                key={section[0].key}
                role="group"
                className={cn(
                  'divide-y divide-foreground/10',
                  index > 0 && 'border-t border-foreground/10',
                )}
              >
                {section.map((item) => (
                  <button
                    key={item.key}
                    type="button"
                    role={item.checked === undefined ? 'menuitem' : 'menuitemradio'}
                    aria-checked={item.checked}
                    disabled={item.disabled}
                    onClick={() => onSelect(item)}
                    className={cn(
                      'flex min-h-[3.25rem] w-full items-center justify-center gap-2.5 px-4 text-base text-primary outline-none active:bg-foreground/[0.08] focus-visible:bg-foreground/[0.06] disabled:pointer-events-none disabled:text-muted-foreground/60',
                      item.destructive && 'text-destructive',
                    )}
                  >
                    {item.icon && (
                      <item.icon
                        className="h-5 w-5 shrink-0"
                      />
                    )}
                    <span
                      className="truncate"
                    >
                      {item.label}
                    </span>
                    {item.checked && (
                      <Check
                        className="h-4 w-4 shrink-0"
                      />
                    )}
                  </button>
                ))}
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={onDismiss}
            className="flex min-h-[3.25rem] w-full items-center justify-center rounded-[14px] bg-popover text-base font-semibold text-primary outline-none active:bg-accent focus-visible:bg-accent"
          >
            Cancel
          </button>
        </div>
      </div>,
      document.body,
    )
  }

  return createPortal(
    <div
      ref={panelRef}
      role="menu"
      aria-label={label}
      onKeyDown={onKeyDown}
      style={
        {
          top: position.top,
          left: position.left,
          right: position.right,
          maxHeight: position.maxHeight,
          '--menu-origin': origin,
        } as React.CSSProperties
      }
      className={cn(
        'motion-menu-in material-card fixed z-50 min-w-[12rem] max-w-[calc(100vw-1rem)] overflow-y-auto rounded-[10px] p-[5px] text-popover-foreground [box-shadow:var(--overlay-shadow)]',
        className,
      )}
    >
      <ExitMarker
        keyframes={fadeOut}
        duration={150}
      />
      {sections.map((section, index) => (
        <div
          key={section[0].key}
          role="group"
          className={cn(index > 0 && 'mt-[5px] border-t border-foreground/10 pt-[5px]')}
        >
          {section.map((item) => (
            <button
              key={item.key}
              type="button"
              role={item.checked === undefined ? 'menuitem' : 'menuitemradio'}
              aria-checked={item.checked}
              disabled={item.disabled}
              onClick={() => onSelect(item)}
              className={cn(
                'group/item flex w-full items-center gap-2.5 rounded-[5px] px-2.5 py-[5px] text-left text-sm outline-none hover:bg-primary hover:text-primary-foreground focus-visible:bg-primary focus-visible:text-primary-foreground disabled:pointer-events-none disabled:opacity-40',
                item.destructive &&
                  'text-destructive hover:bg-destructive hover:text-destructive-foreground focus-visible:bg-destructive focus-visible:text-destructive-foreground',
              )}
            >
              {item.icon && (
                <item.icon
                  className="h-4 w-4 shrink-0 opacity-80"
                />
              )}
              <span
                className="min-w-0 flex-1 truncate"
              >
                {item.label}
              </span>
              {item.checked && (
                <Check
                  className="h-4 w-4 shrink-0"
                />
              )}
            </button>
          ))}
        </div>
      ))}
    </div>,
    document.body,
  )
}

export interface ContextMenuPoint {
  x: number
  y: number
}

export interface ContextMenuProps {
  label: string
  sections: MenuItem[][]
  point: ContextMenuPoint | null
  onClose: () => void
}

export function ContextMenu({ label, sections, point, onClose }: ContextMenuProps) {
  const [position, setPosition] = useState<MenuPosition>({ top: 0, left: 0 })
  const menuRef = useRef<HTMLDivElement>(null)
  const visibleSections = sections.filter((section) => section.length > 0)

  useLayoutEffect(() => {
    if (!point) {
      return
    }

    const rect = menuRef.current?.getBoundingClientRect()
    const width = rect?.width ?? 0
    const height = rect?.height ?? 0
    setPosition({
      top: Math.max(8, Math.min(point.y, window.innerHeight - height - 8)),
      left: Math.max(8, Math.min(point.x, window.innerWidth - width - 8)),
    })
  }, [point])

  useEffect(() => {
    if (!point) {
      return
    }

    menuItems(menuRef.current)[0]?.focus()
    const handlePointerDown = (event: PointerEvent) => {
      if (!menuRef.current?.contains(event.target as Node)) {
        onClose()
      }
    }
    const sheet = matches(PHONE_QUERY)
    document.addEventListener('pointerdown', handlePointerDown)
    window.addEventListener('resize', onClose)
    if (!sheet) {
      window.addEventListener('scroll', onClose, true)
    }
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      window.removeEventListener('resize', onClose)
      window.removeEventListener('scroll', onClose, true)
    }
  }, [point, onClose])

  if (!point) {
    return null
  }

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'Escape' || event.key === 'Tab') {
      event.preventDefault()
      onClose()
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      moveFocus(menuRef.current, 1)
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      moveFocus(menuRef.current, -1)
    }
  }

  return (
    <MenuPanel
      panelRef={menuRef}
      label={label}
      sections={visibleSections}
      position={position}
      origin="top left"
      onKeyDown={handleKeyDown}
      onSelect={(item) => {
        onClose()
        item.onSelect()
      }}
      onDismiss={onClose}
    />
  )
}
