import * as React from 'react'
import { Search, XCircle } from 'lucide-react'
import { cn, coarsePointer } from '@/lib/utils'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const fieldClasses = cn(
  'flex h-8 w-full rounded-[7px] border-0 bg-card px-2.5 py-1.5 text-sm text-foreground [box-shadow:inset_0_0_0_1px_hsl(var(--input)),0_0.5px_1px_hsl(0_0%_0%/0.05)] placeholder:text-muted-foreground/60 transition-shadow dark:bg-foreground/[0.05]',
  'file:border-0 file:bg-transparent file:text-sm file:font-medium',
  'focus-visible:outline-none focus-visible:[box-shadow:inset_0_0_0_1px_hsl(var(--ring)),0_0_0_3.5px_hsl(var(--ring)/0.35)]',
  'aria-[invalid=true]:[box-shadow:inset_0_0_0_1px_hsl(var(--destructive))]',
  'disabled:cursor-not-allowed disabled:opacity-50',
)

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, autoFocus, ...props }, ref) => (
    <input
      type={type}
      className={cn(fieldClasses, className)}
      ref={ref}
      autoFocus={autoFocus && !coarsePointer()}
      {...props}
    />
  ),
)
Input.displayName = 'Input'

export interface SearchInputProps extends Omit<InputProps, 'type'> {
  onClear?: () => void
}

const SearchInput = React.forwardRef<HTMLInputElement, SearchInputProps>(
  ({ className, autoFocus, onClear, value, ...props }, ref) => (
    <div
      className={cn('relative flex w-full items-center', className)}
    >
      <Search
        aria-hidden="true"
        className="pointer-events-none absolute left-2 h-3.5 w-3.5 text-muted-foreground"
      />
      <input
        type="search"
        ref={ref}
        value={value}
        autoFocus={autoFocus && !coarsePointer()}
        className={cn(
          fieldClasses,
          'rounded-full pl-7 [&::-webkit-search-cancel-button]:appearance-none',
          onClear && 'pr-7',
        )}
        {...props}
      />
      {onClear && value !== undefined && value !== '' && (
        <button
          type="button"
          aria-label="Clear Search"
          onClick={onClear}
          className="absolute right-1.5 flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground/70 hover:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <XCircle
            aria-hidden="true"
            className="h-3.5 w-3.5"
          />
        </button>
      )}
    </div>
  ),
)
SearchInput.displayName = 'SearchInput'

export { Input, SearchInput }
