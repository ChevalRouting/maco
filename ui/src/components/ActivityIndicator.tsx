import { cn } from '@/lib/utils'

const SPOKES = Array.from({ length: 12 }, (_, index) => index)

export function ActivityIndicator({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className={cn('activity-indicator h-4 w-4', className)}
    >
      {SPOKES.map((index) => (
        <rect
          key={index}
          x="11"
          y="1.5"
          width="2"
          height="6.5"
          rx="1"
          fill="currentColor"
          opacity={0.2 + (0.8 * index) / 11}
          transform={`rotate(${index * 30} 12 12)`}
        />
      ))}
    </svg>
  )
}
