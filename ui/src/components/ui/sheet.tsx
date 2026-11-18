import { useEffect, useRef, useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import { cn, coarsePointer } from '@/lib/utils'
import { COMPACT_QUERY, EASE_SHEET, ExitMarker, currentTransform, fadeOut, matches } from '@/lib/motion'

interface SheetProps {
  open: boolean
  onClose: () => void
  title: React.ReactNode
  children: React.ReactNode
  className?: string
}

const DISMISS_DISTANCE = 120
const DISMISS_VELOCITY = 0.6

function sheetExit(node: HTMLElement): Keyframe[] {
  if (matches(COMPACT_QUERY)) {
    return [{ transform: currentTransform(node) }, { transform: 'translateY(100%)' }]
  }
  return [
    { opacity: 1, transform: 'none' },
    { opacity: 0, transform: 'translateY(-8px) scale(0.98)' },
  ]
}

interface DragState {
  start: number
  last: number
  time: number
  velocity: number
}

function useDragToDismiss(content: React.RefObject<HTMLDivElement>, onDismiss: () => void) {
  const drag = useRef<DragState | null>(null)

  const onPointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!matches(COMPACT_QUERY) || (event.target as HTMLElement).closest('button')) {
      return
    }
    event.currentTarget.setPointerCapture(event.pointerId)
    drag.current = { start: event.clientY, last: event.clientY, time: event.timeStamp, velocity: 0 }
  }

  const onPointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    const state = drag.current
    const node = content.current
    if (!state || !node) {
      return
    }
    const elapsed = Math.max(1, event.timeStamp - state.time)
    state.velocity = (event.clientY - state.last) / elapsed
    state.last = event.clientY
    state.time = event.timeStamp
    const offset = event.clientY - state.start
    const distance = offset > 0 ? offset : offset / 6
    node.style.animation = 'none'
    node.style.transform = `translateY(${distance}px)`
  }

  const onPointerUp = () => {
    const state = drag.current
    const node = content.current
    drag.current = null
    if (!state || !node) {
      return
    }
    const offset = state.last - state.start
    if (offset > DISMISS_DISTANCE || (offset > 24 && state.velocity > DISMISS_VELOCITY)) {
      onDismiss()
      return
    }
    const from = node.style.transform
    node.style.transform = ''
    if (from) {
      node.animate([{ transform: from }, { transform: 'translateY(0)' }], { duration: 260, easing: EASE_SHEET })
    }
  }

  return { onPointerDown, onPointerMove, onPointerUp, onPointerCancel: onPointerUp }
}

export function Sheet({ open, onClose, title, children, className }: SheetProps) {
  const content = useRef<HTMLDivElement>(null)
  const drag = useDragToDismiss(content, onClose)
  const compact = matches(COMPACT_QUERY)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    if (!open) {
      setReady(false)
      return
    }
    const frame = requestAnimationFrame(() => setReady(true))
    return () => cancelAnimationFrame(frame)
  }, [open])

  useEffect(() => {
    const node = content.current
    if (!ready || !node || coarsePointer() || document.activeElement !== node) {
      return
    }
    node.querySelector<HTMLElement>('input:not([type="hidden"]):not(:disabled), textarea:not(:disabled), [role="combobox"]')?.focus()
  }, [ready])

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(o) => {
        if (!o) {
          onClose()
        }
      }}
    >
      <Dialog.Portal>
        <div
          aria-hidden="true"
          className="motion-fade-in fixed inset-0 z-40 bg-black/25 md:bg-black/15"
        >
          <ExitMarker
            keyframes={fadeOut}
            duration={220}
          />
        </div>
        <Dialog.Content
          ref={content}
          aria-describedby={undefined}
          className={cn(
            'motion-sheet-in fixed inset-x-0 bottom-0 top-[calc(env(safe-area-inset-top,0px)+0.625rem)] z-40 flex flex-col overflow-hidden rounded-t-[14px] bg-background [box-shadow:var(--overlay-shadow)] focus:outline-none',
            'md:bottom-auto md:top-14 md:mx-auto md:max-h-[calc(100dvh-6rem)] md:w-[calc(100%-4rem)] md:max-w-2xl md:rounded-2xl',
            className,
          )}
          onOpenAutoFocus={(e) => {
            e.preventDefault()
            if (!coarsePointer()) {
              content.current?.focus()
            }
          }}
        >
          <ExitMarker
            keyframes={sheetExit}
            duration={compact ? 320 : 200}
            easing={compact ? EASE_SHEET : undefined}
          />
          <div
            className="shrink-0 touch-none select-none border-b border-border/60 bg-sidebar md:touch-auto"
            onPointerDown={drag.onPointerDown}
            onPointerMove={drag.onPointerMove}
            onPointerUp={drag.onPointerUp}
            onPointerCancel={drag.onPointerCancel}
          >
            <div
              aria-hidden="true"
              className="flex justify-center pt-1.5 md:hidden"
            >
              <div
                className="h-[5px] w-9 rounded-full bg-muted-foreground/30"
              />
            </div>
            <div
              className="grid h-11 grid-cols-[1.75rem_minmax(0,1fr)_1.75rem] items-center gap-3 px-4 md:h-12"
            >
              <span
                aria-hidden="true"
              />
              <Dialog.Title
                className="truncate text-center text-sm font-semibold"
              >
                {title}
              </Dialog.Title>
              <Dialog.Close
                asChild
              >
                <button
                  type="button"
                  aria-label="Close"
                  title="Close"
                  className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-foreground/[0.07] text-muted-foreground hover:bg-foreground/[0.12] hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <X
                    strokeWidth={2.5}
                    className="h-3.5 w-3.5"
                  />
                </button>
              </Dialog.Close>
            </div>
          </div>
          <div
            className="min-h-0 flex-1 space-y-5 overflow-y-auto overscroll-contain px-4 pb-[calc(env(safe-area-inset-bottom,0px)+1.25rem)] pt-5 md:px-5 md:pb-5"
          >
            {ready && children}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
