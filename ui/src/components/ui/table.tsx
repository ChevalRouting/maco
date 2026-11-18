import * as React from 'react'
import { cn } from '@/lib/utils'

export interface TableProps extends React.HTMLAttributes<HTMLTableElement> {
  cards?: boolean
}

function labelCells(table: HTMLTableElement) {
  const headRow = table.tHead?.rows[table.tHead.rows.length - 1]
  if (!headRow) {
    return
  }
  const labels: string[] = []
  for (const th of Array.from(headRow.cells)) {
    const text = th.textContent?.trim() ?? ''
    for (let i = 0; i < th.colSpan; i++) {
      labels.push(text)
    }
  }
  for (const body of Array.from(table.tBodies)) {
    for (const row of Array.from(body.rows)) {
      let column = 0
      for (const cell of Array.from(row.cells)) {
        const label = cell.colSpan > 1 ? '' : (labels[column] ?? '')
        if (label && cell.dataset.label !== label) {
          cell.dataset.label = label
        } else if (!label && cell.dataset.label !== undefined) {
          delete cell.dataset.label
        }
        column += cell.colSpan
      }
    }
  }
}

const Table = React.forwardRef<HTMLTableElement, TableProps>(
  ({ className, cards = true, ...props }, ref) => {
    const inner = React.useRef<HTMLTableElement>(null)
    React.useImperativeHandle(ref, () => inner.current as HTMLTableElement)

    React.useLayoutEffect(() => {
      const table = inner.current
      if (!table || !cards) {
        return
      }
      labelCells(table)
      const observer = new MutationObserver(() => labelCells(table))
      observer.observe(table, { childList: true, subtree: true, characterData: true })
      return () => observer.disconnect()
    }, [cards])

    return (
      <div
        className="relative w-full overflow-auto rounded-xl bg-card [box-shadow:var(--card-shadow)]"
      >
        <table
          ref={inner}
          data-cards={cards ? '' : undefined}
          className={cn(
            'w-full caption-bottom text-sm [&_td:first-child]:pl-4 [&_td:last-child]:pr-4 [&_th:first-child]:pl-4 [&_th:last-child]:pr-4',
            className,
          )}
          {...props}
        />
      </div>
    )
  },
)
Table.displayName = 'Table'

const TableHeader = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <thead
    ref={ref}
    className={cn('[&_tr]:border-b [&_tr]:border-border/80', className)}
    {...props}
  />
))
TableHeader.displayName = 'TableHeader'

const TableBody = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tbody
    ref={ref}
    className={cn('[&_tr:last-child]:border-0', className)}
    {...props}
  />
))
TableBody.displayName = 'TableBody'

const TableFooter = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tfoot
    ref={ref}
    className={cn(
      'border-t border-border/80 font-medium [&>tr]:last:border-b-0',
      className,
    )}
    {...props}
  />
))
TableFooter.displayName = 'TableFooter'

const TableRow = React.forwardRef<HTMLTableRowElement, React.HTMLAttributes<HTMLTableRowElement>>(
  ({ className, ...props }, ref) => (
    <tr
      ref={ref}
      className={cn(
        'border-b border-border/60 data-[state=selected]:bg-primary data-[state=selected]:text-primary-foreground',
        className,
      )}
      {...props}
    />
  ),
)
TableRow.displayName = 'TableRow'

const TableHead = React.forwardRef<
  HTMLTableCellElement,
  React.ThHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <th
    ref={ref}
    className={cn(
      'h-8 px-3 text-left align-middle text-xs font-medium text-muted-foreground [&:has([role=checkbox])]:pr-0',
      className,
    )}
    {...props}
  />
))
TableHead.displayName = 'TableHead'

const TableCell = React.forwardRef<
  HTMLTableCellElement,
  React.TdHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <td
    ref={ref}
    className={cn('px-3 py-2 align-middle text-sm [&:has([role=checkbox])]:pr-0', className)}
    {...props}
  />
))
TableCell.displayName = 'TableCell'

const TableCaption = React.forwardRef<
  HTMLTableCaptionElement,
  React.HTMLAttributes<HTMLTableCaptionElement>
>(({ className, ...props }, ref) => (
  <caption
    ref={ref}
    className={cn('mt-3 text-xs text-muted-foreground', className)}
    {...props}
  />
))
TableCaption.displayName = 'TableCaption'

export { Table, TableHeader, TableBody, TableFooter, TableHead, TableRow, TableCell, TableCaption }
