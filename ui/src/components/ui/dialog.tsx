import { useRef } from 'react'
import * as RDialog from '@radix-ui/react-dialog'
import { TriangleAlert, X } from 'lucide-react'
import { cn, coarsePointer } from '@/lib/utils'
import { ExitMarker, fadeOut } from '@/lib/motion'
import { Button } from './button'

interface DialogProps {
  open: boolean
  onClose: () => void
  title?: React.ReactNode
  description?: React.ReactNode
  children?: React.ReactNode
  footer?: React.ReactNode
  className?: string
  hideClose?: boolean
  captureKeyboard?: boolean
  alert?: boolean
  icon?: React.ReactNode
}

function dialogExit(): Keyframe[] {
  return [
    { opacity: 1, transform: 'scale(1)' },
    { opacity: 0, transform: 'scale(0.97)' },
  ]
}

export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  className,
  hideClose,
  captureKeyboard,
  alert,
  icon,
}: DialogProps) {
  const previousFocus = useRef<HTMLElement | null>(null)
  const wasOpen = useRef(false)
  if (open && !wasOpen.current && typeof document !== 'undefined') {
    previousFocus.current = document.activeElement as HTMLElement
  }
  wasOpen.current = open
  return (
    <RDialog.Root
      open={open}
      onOpenChange={(o) => {
        if (!o) {
          onClose()
        }
      }}
    >
      <RDialog.Portal>
        <div
          aria-hidden="true"
          className="motion-fade-in fixed inset-0 z-50 bg-black/25"
        >
          <ExitMarker
            keyframes={fadeOut}
            duration={180}
          />
        </div>
        <RDialog.Content
          {...(!description ? { 'aria-describedby': undefined } : {})}
          onOpenAutoFocus={(event) => {
            if (coarsePointer()) {
              event.preventDefault()
            }
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            previousFocus.current?.focus()
          }}
          onEscapeKeyDown={(event) => {
            if (captureKeyboard && (event.target as HTMLElement).closest('[data-console-input]')) {
              event.preventDefault()
            }
          }}
          className={cn(
            'motion-pop-in fixed inset-0 z-50 m-auto flex h-fit max-h-[90dvh] w-[calc(100vw-2rem)] flex-col rounded-2xl bg-background [box-shadow:var(--overlay-shadow)] focus:outline-none',
            className ?? 'max-w-lg',
          )}
        >
          <ExitMarker
            keyframes={dialogExit}
            duration={160}
          />
          {(title || description || !hideClose) && (
            <div
              className={cn(
                'flex items-start gap-3 shrink-0',
                alert ? 'flex-col items-center px-6 pb-1 pt-6 text-center' : 'px-5 py-4',
              )}
            >
              {icon}
              <div
                className="min-w-0 flex-1"
              >
                {title && (
                  <RDialog.Title
                    className={cn('font-semibold tracking-tight', alert ? 'text-sm font-bold' : 'text-[15px]')}
                  >
                    {title}
                  </RDialog.Title>
                )}
                {description && (
                  <RDialog.Description
                    className={cn('text-muted-foreground', alert ? 'mt-1.5 text-xs leading-snug' : 'mt-0.5 text-sm')}
                  >
                    {description}
                  </RDialog.Description>
                )}
                {!title && (
                  <RDialog.Title
                    className="sr-only"
                  >
                    Dialog
                  </RDialog.Title>
                )}
              </div>
              {!hideClose && (
                <RDialog.Close
                  asChild
                >
                  <button
                    type="button"
                    className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-foreground/[0.07] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring text-muted-foreground hover:bg-foreground/[0.12] hover:text-foreground"
                    aria-label="Close"
                  >
                    <X
                      className="h-4 w-4"
                    />
                  </button>
                </RDialog.Close>
              )}
            </div>
          )}
          {children != null && (
            <div
              className="min-h-0 flex-1 overflow-y-auto px-5 py-5"
            >
              {children}
            </div>
          )}
          {footer && (
            <div
              className={cn(
                'flex shrink-0 flex-wrap items-center justify-end gap-2',
                alert ? 'grid grid-cols-2 px-5 pb-5 pt-4 [&>*]:w-full' : 'px-5 pb-4 pt-2',
              )}
            >
              {footer}
            </div>
          )}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

interface AlertDialogProps {
  open: boolean
  onCancel: () => void
  onConfirm: () => void
  title: React.ReactNode
  description?: React.ReactNode
  confirmLabel?: string
  cancelLabel?: string
  destructive?: boolean
  busy?: boolean
}

export function AlertDialog({
  open,
  onCancel,
  onConfirm,
  title,
  description,
  confirmLabel = 'Continue',
  cancelLabel = 'Cancel',
  destructive,
  busy,
}: AlertDialogProps) {
  return (
    <Dialog
      open={open}
      onClose={() => {
        if (!busy) {
          onCancel()
        }
      }}
      title={title}
      description={description}
      className="max-w-[18.5rem]"
      hideClose
      alert
      icon={(
        <span
          className={cn(
            'grid h-12 w-12 place-items-center rounded-[0.875rem] bg-card [box-shadow:var(--card-shadow)]',
            destructive ? 'text-warning' : 'text-primary',
          )}
        >
          <TriangleAlert
            aria-hidden="true"
            className="h-7 w-7"
          />
        </span>
      )}
      footer={(
        <>
          <Button
            onClick={onCancel}
            disabled={busy}
            autoFocus={destructive}
          >
            {cancelLabel}
          </Button>
          <Button
            variant={destructive ? 'destructive' : 'suggested'}
            onClick={onConfirm}
            disabled={busy}
            autoFocus={!destructive}
          >
            {confirmLabel}
          </Button>
        </>
      )}
    />
  )
}
