import { ActivityIndicator } from '@/components/ActivityIndicator'
import { cn } from '@/lib/utils'

interface SpinnerProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

const SIZE = { sm: 'h-4 w-4', md: 'h-5 w-5', lg: 'h-6 w-6' }

export function Spinner({ size = 'md', className }: SpinnerProps) {
  return (
    <div
      className={cn('flex items-center justify-center h-64', className)}
    >
      <ActivityIndicator
        className={cn(SIZE[size], 'text-muted-foreground')}
      />
    </div>
  )
}
