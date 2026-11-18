import { type ReactNode } from 'react'

interface StableWidthProps {
  value: string
  values: readonly string[]
  render?: (value: string) => ReactNode
  align?: 'start' | 'center' | 'end'
}

const alignments = {
  start: 'justify-items-start',
  center: 'justify-items-center',
  end: 'justify-items-end',
}

export function StableWidth({ value, values, render = (option) => option, align = 'center' }: StableWidthProps) {
  const options = values.includes(value) ? values : [value, ...values]
  return (
    <span className={`inline-grid ${alignments[align]}`}>
      {options.map((option) => (
        <span key={option} className={option === value ? 'col-start-1 row-start-1' : 'invisible col-start-1 row-start-1'}>
          {render(option)}
        </span>
      ))}
    </span>
  )
}
