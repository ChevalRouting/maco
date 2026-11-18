import * as React from 'react'
import { cn } from '@/lib/utils'

interface SwitchProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}

export function Switch({
  checked,
  onCheckedChange,
  disabled,
  id,
  className,
  ...props
}: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      id={id}
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      {...props}
      className={cn(
        'relative inline-flex h-6 w-[2.625rem] shrink-0 items-center rounded-full transition-colors duration-200 sm:h-[1.375rem] sm:w-[2.375rem]',
        'focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50',
        'disabled:cursor-not-allowed disabled:opacity-50',
        checked
          ? 'bg-primary [box-shadow:inset_0_0_0_0.5px_hsl(0_0%_0%/0.12)]'
          : 'bg-foreground/[0.09] [box-shadow:inset_0_0_0_0.5px_hsl(0_0%_0%/0.1)] dark:bg-foreground/[0.16]',
        className,
      )}
    >
      <span
        className={cn(
          'pointer-events-none block h-5 w-5 rounded-full bg-[hsl(var(--switch-thumb))] shadow-[0_0_0_0.5px_hsl(0_0%_0%/0.06),0_1px_2.5px_hsl(0_0%_0%/0.25)] transition-transform duration-200 ease-out sm:h-[1.125rem] sm:w-[1.125rem]',
          checked ? 'translate-x-5 sm:translate-x-[1.125rem]' : 'translate-x-0.5',
        )}
      />
    </button>
  )
}
