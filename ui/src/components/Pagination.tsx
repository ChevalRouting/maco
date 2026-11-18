import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface PaginationProps {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPage: (page: number) => void
  unit?: string
  className?: string
}

export function Pagination({
  page,
  totalPages,
  total,
  pageSize,
  onPage,
  unit = 'items',
  className,
}: PaginationProps) {
  if (totalPages <= 1) {
    return null
  }

  const from = page * pageSize + 1
  const to = Math.min((page + 1) * pageSize, total)

  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground',
        className,
      )}
    >
      <span>
        {`${from.toLocaleString()}–${to.toLocaleString()} of ${total.toLocaleString()} ${unit}`}
      </span>
      <div
        className="flex items-center gap-2"
      >
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          disabled={page === 0}
          onClick={() => onPage(page - 1)}
          title="Previous page"
        >
          <ChevronLeft
            className="h-4 w-4"
          />
        </Button>
        <span
          className="tabular-nums"
        >
          {`Page ${page + 1} / ${totalPages}`}
        </span>
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          disabled={page >= totalPages - 1}
          onClick={() => onPage(page + 1)}
          title="Next page"
        >
          <ChevronRight
            className="h-4 w-4"
          />
        </Button>
      </div>
    </div>
  )
}
