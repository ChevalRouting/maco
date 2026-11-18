import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-lg text-sm font-medium transition-[background-color,box-shadow,filter] duration-150 select-none active:brightness-95 focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none disabled:opacity-40',
  {
    variants: {
      variant: {
        default:
          'bg-secondary text-secondary-foreground hover:brightness-[0.97] dark:hover:brightness-110 [box-shadow:var(--control-shadow)]',
        suggested:
          'bg-primary bg-gradient-to-b from-white/[0.07] to-transparent text-primary-foreground hover:brightness-105 shadow-[0_0_0_0.5px_hsl(var(--primary)/0.7),0_1px_1.5px_hsl(0_0%_0%/0.15)]',
        destructive:
          'bg-destructive bg-gradient-to-b from-white/[0.07] to-transparent text-destructive-foreground hover:brightness-105 shadow-[0_0_0_0.5px_hsl(var(--destructive)/0.7),0_1px_1.5px_hsl(0_0%_0%/0.15)]',
        outline:
          'bg-secondary text-secondary-foreground hover:brightness-[0.97] dark:hover:brightness-110 [box-shadow:var(--control-shadow)]',
        secondary:
          'bg-foreground/[0.07] text-foreground hover:bg-foreground/[0.1] active:bg-foreground/[0.14] active:brightness-100',
        ghost:
          'text-primary hover:text-primary/80 active:text-primary/60 active:brightness-100',
        link: 'text-primary underline-offset-4 hover:underline p-0 h-auto',
      },
      size: {
        default: 'h-8 px-3.5 py-1.5',
        sm: 'h-7 rounded-md px-2.5 text-xs',
        lg: 'h-10 rounded-[10px] px-6 text-sm',
        icon: 'h-8 w-8',
      },
    },
    compoundVariants: [
      {
        variant: 'ghost',
        size: 'icon',
        className:
          'text-foreground/70 hover:bg-foreground/[0.07] hover:text-foreground active:bg-foreground/[0.12] active:text-foreground',
      },
    ],
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  },
)
Button.displayName = 'Button'

export { Button, buttonVariants }
