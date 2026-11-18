import * as React from 'react'
import { ChevronRight, Eye, EyeOff } from 'lucide-react'
import { cn, coarsePointer } from '@/lib/utils'
import { Switch } from '@/components/ui/switch'

export function PreferencesGroup({
  title,
  description,
  header,
  children,
  className,
}: {
  title?: string
  description?: string
  header?: React.ReactNode
  children: React.ReactNode
  className?: string
}) {
  return (
    <section
      className={cn('space-y-2', className)}
    >
      {(title || description || header) && (
        <div
          className="flex flex-wrap items-end justify-between gap-3 px-1"
        >
          <div
            className="min-w-0 flex-1 basis-[min(100%,14rem)]"
          >
            {title && (
              <h2
                className="text-sm font-semibold text-foreground"
              >
                {title}
              </h2>
            )}
            {description && (
              <p
                className="mt-0.5 text-xs text-muted-foreground"
              >
                {description}
              </p>
            )}
          </div>
          {header && (
            <div
              className="min-w-0 max-w-full"
            >
              {header}
            </div>
          )}
        </div>
      )}
      <div
        className="inset-separators overflow-hidden rounded-xl bg-card [box-shadow:var(--card-shadow)]"
      >
        {children}
      </div>
    </section>
  )
}

interface RowProps {
  title?: React.ReactNode
  subtitle?: React.ReactNode
  prefix?: React.ReactNode
  children?: React.ReactNode
  onClick?: () => void
  className?: string
}

export function Row({ title, subtitle, prefix, children, onClick, className }: RowProps) {
  const body = (
    <>
      {prefix && (
        <span
          className="shrink-0 text-muted-foreground"
        >
          {prefix}
        </span>
      )}
      <div
        className="min-w-0 flex-1 basis-[min(100%,14rem)]"
      >
        {title != null && (
          <div
            className="text-sm text-foreground break-words"
          >
            {title}
          </div>
        )}
        {subtitle != null && (
          <div
            className="mt-0.5 text-xs text-muted-foreground break-words"
          >
            {subtitle}
          </div>
        )}
      </div>
      {children != null && (
        <div
          className="flex min-w-0 max-w-full flex-wrap items-center gap-2"
        >
          {children}
        </div>
      )}
    </>
  )
  const base = cn('flex flex-wrap min-h-[2.75rem] w-full gap-x-3 gap-y-1.5 px-4 py-2', className)
  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(base, 'w-full items-center text-left focus-visible:outline-none focus-visible:[box-shadow:inset_0_0_0_2px_hsl(var(--ring)/0.55)] active:bg-foreground/[0.06]')}
      >
        {body}
        <ChevronRight
          strokeWidth={2.5}
          className="h-3.5 w-3.5 shrink-0 self-center text-muted-foreground/50"
        />
      </button>
    )
  }
  return (
    <div
      className={cn(base, 'items-center')}
    >
      {body}
    </div>
  )
}

export function ComboRow({
  title,
  subtitle,
  children,
}: {
  title: React.ReactNode
  subtitle?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div
      className="grid min-h-[2.75rem] min-w-0 items-center gap-x-3 gap-y-1.5 px-4 py-2 grid-cols-[repeat(auto-fit,minmax(min(100%,14rem),1fr))]"
    >
      <div
        className="min-w-0"
      >
        <div
          className="break-words text-sm text-foreground"
        >
          {title}
        </div>
        {subtitle != null && (
          <div
            className="mt-0.5 break-words text-xs text-muted-foreground"
          >
            {subtitle}
          </div>
        )}
      </div>
      <div
        className="min-w-0 w-full"
      >
        {children}
      </div>
    </div>
  )
}

export interface EntryRowProps extends React.InputHTMLAttributes<HTMLInputElement> {
  title: string
  suffix?: React.ReactNode
  error?: string | null
  help?: React.ReactNode
  stacked?: boolean
}

export const EntryRow = React.forwardRef<HTMLInputElement, EntryRowProps>(
  ({ title, suffix, error, help, stacked, className, ...props }, ref) => {
    const generatedId = React.useId()
    const id = props.id || generatedId
    const [visible, setVisible] = React.useState(false)
    const inputRef = React.useRef<HTMLInputElement | null>(null)
    const setInputRef = React.useCallback(
      (input: HTMLInputElement | null) => {
        inputRef.current = input
        if (typeof ref === 'function') {
          ref(input)
        } else if (ref) {
          ref.current = input
        }
      },
      [ref],
    )
    React.useEffect(() => {
      inputRef.current?.setCustomValidity(error || '')
    }, [error])
    return (
      <div
        className={cn(
          'min-w-0 gap-x-3 gap-y-1.5 px-4 py-2',
          stacked
            ? 'flex flex-col'
            : 'grid min-h-[2.75rem] items-center grid-cols-[repeat(auto-fit,minmax(min(100%,14rem),1fr))]',
        )}
      >
        <label
          htmlFor={id}
          className={cn(
            'block break-words',
            stacked
              ? 'text-xs font-medium'
              : 'flex min-h-8 items-center self-start text-sm',
            error ? 'text-destructive' : stacked ? 'text-muted-foreground' : 'text-foreground',
          )}
        >
          {title}
        </label>
        <div
          className="flex min-w-0 flex-col gap-1.5"
        >
          <div
            className="flex min-w-0 items-center gap-2"
          >
            <div
              className={cn(
                'flex h-8 min-w-0 flex-1 items-center gap-1 rounded-[7px] bg-card px-2.5 transition-shadow dark:bg-foreground/[0.05]',
                error
                  ? '[box-shadow:inset_0_0_0_1px_hsl(var(--destructive))]'
                  : '[box-shadow:inset_0_0_0_1px_hsl(var(--input)),0_0.5px_1px_hsl(0_0%_0%/0.05)]',
                'focus-within:[box-shadow:inset_0_0_0_1px_hsl(var(--ring)),0_0_0_3.5px_hsl(var(--ring)/0.35)]',
                props.disabled && 'opacity-50',
              )}
            >
              <input
                ref={setInputRef}
                aria-invalid={error ? true : undefined}
                className={cn(
                  'min-w-0 w-full flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground/60 disabled:cursor-not-allowed',
                  className,
                )}
                {...props}
                autoFocus={props.autoFocus && !coarsePointer()}
                id={id}
                type={props.type === 'password' && visible ? 'text' : props.type}
                aria-describedby={
                  [props['aria-describedby'], help && `${id}-help`, error && `${id}-error`]
                    .filter(Boolean)
                    .join(' ') || undefined
                }
              />
              {props.type === 'password' && (
                <button
                  type="button"
                  disabled={props.disabled}
                  aria-label={visible ? 'Hide Password' : 'Show Password'}
                  aria-pressed={visible}
                  onClick={() => setVisible(!visible)}
                  className="-mr-1 flex h-6 w-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {visible ? (
                    <EyeOff
                      aria-hidden="true"
                      className="h-3.5 w-3.5"
                    />
                  ) : (
                    <Eye
                      aria-hidden="true"
                      className="h-3.5 w-3.5"
                    />
                  )}
                </button>
              )}
            </div>
            {suffix && (
              <div
                className="flex shrink-0 items-center gap-1"
              >
                {suffix}
              </div>
            )}
          </div>
          {help && (
            <p
              id={`${id}-help`}
              className="text-xs text-muted-foreground break-words"
            >
              {help}
            </p>
          )}
          {error && (
            <p
              id={`${id}-error`}
              className="text-xs text-destructive break-words"
            >
              {error}
            </p>
          )}
        </div>
      </div>
    )
  },
)
EntryRow.displayName = 'EntryRow'

export function SwitchRow({
  title,
  subtitle,
  checked,
  onCheckedChange,
  disabled,
}: {
  title: React.ReactNode
  subtitle?: React.ReactNode
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
}) {
  const titleId = React.useId()

  return (
    <div
      onClick={(event) => {
        if (disabled || (event.target as HTMLElement).closest('[role="switch"]')) {
          return
        }
        onCheckedChange(!checked)
      }}
      className={cn(
        'flex min-h-[2.75rem] w-full items-center gap-3 px-4 py-2 text-left',
        disabled ? 'opacity-50' : 'cursor-default',
      )}
    >
      <div
        className="min-w-0 flex-1"
      >
        <div
          id={titleId}
          className="text-sm text-foreground break-words"
        >
          {title}
        </div>
        {subtitle != null && (
          <div
            id={`${titleId}-help`}
            className="mt-0.5 text-xs text-muted-foreground break-words"
          >
            {subtitle}
          </div>
        )}
      </div>
      <Switch
        className="shrink-0"
        aria-label={typeof title === 'string' ? title : undefined}
        aria-labelledby={titleId}
        aria-describedby={subtitle != null ? `${titleId}-help` : undefined}
        checked={checked}
        onCheckedChange={onCheckedChange}
        disabled={disabled}
      />
    </div>
  )
}
