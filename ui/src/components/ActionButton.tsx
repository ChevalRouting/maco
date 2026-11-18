import { forwardRef } from 'react'
import { type LucideIcon } from 'lucide-react'
import { ActivityIndicator } from '@/components/ActivityIndicator'
import { Button, type ButtonProps } from '@/components/ui/button'

export interface ActionButtonProps extends Omit<ButtonProps, 'children' | 'asChild'> {
  label: string
  icon: LucideIcon
  busy?: boolean
}

export const ActionButton = forwardRef<HTMLButtonElement, ActionButtonProps>(
  ({ label, icon: Icon, busy, disabled, variant = 'ghost', size = 'icon', ...props }, ref) => (
    <span
      title={label}
      className="inline-flex"
    >
      <Button
        {...props}
        ref={ref}
        type={props.type ?? 'button'}
        variant={variant}
        size={size}
        aria-label={label}
        aria-busy={busy || undefined}
        disabled={disabled || busy}
      >
        {busy ? (
          <ActivityIndicator
            className="h-4 w-4"
          />
        ) : (
          <Icon
            aria-hidden="true"
            className="h-4 w-4"
          />
        )}
      </Button>
    </span>
  ),
)
ActionButton.displayName = 'ActionButton'
