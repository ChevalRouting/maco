import * as React from 'react'
import { cn, coarsePointer } from '@/lib/utils'

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, autoFocus, ...props }, ref) => {
    return (
      <textarea
        className={cn(
          'flex min-h-[80px] w-full rounded-[7px] border-0 bg-card px-2.5 py-1.5 text-sm text-foreground [box-shadow:inset_0_0_0_1px_hsl(var(--input)),0_0.5px_1px_hsl(0_0%_0%/0.05)] placeholder:text-muted-foreground/60 transition-shadow focus-visible:outline-none focus-visible:[box-shadow:inset_0_0_0_1px_hsl(var(--ring)),0_0_0_3.5px_hsl(var(--ring)/0.35)] disabled:cursor-not-allowed disabled:opacity-50 dark:bg-foreground/[0.05]',
          className,
        )}
        ref={ref}
        autoFocus={autoFocus && !coarsePointer()}
        {...props}
      />
    )
  },
)
Textarea.displayName = 'Textarea'

export { Textarea }
