import { useEffect, useState } from 'react'
import { ValidatedEntryRow } from './ValidatedEntryRow'

interface MemoryEntryRowProps {
  id: string
  value: number
  onValueChange: (mib: number) => void
  disabled?: boolean
}

const decimal = /^\d+(\.\d+)?$/

export function MemoryEntryRow({ id, value, onValueChange, disabled }: MemoryEntryRowProps) {
  const [text, setText] = useState(String(value / 1024))

  useEffect(() => {
    setText(String(value / 1024))
  }, [value])

  const mib = Number(text) * 1024
  const error =
    decimal.test(text) && Number.isSafeInteger(mib) && mib >= 64
      ? ''
      : 'Enter at least 0.0625 GiB (64 MiB), in increments of 1 MiB.'

  return (
    <ValidatedEntryRow
      id={id}
      title="Memory (GiB)"
      required
      disabled={disabled}
      inputMode="decimal"
      value={text}
      error={error}
      onChange={(event) => {
        const next = event.target.value
        setText(next)
        const amount = Number(next) * 1024
        if (decimal.test(next) && Number.isSafeInteger(amount) && amount >= 64) {
          onValueChange(amount)
        }
      }}
    />
  )
}
