import { useEffect, useState } from 'react'
import { ValidatedEntryRow } from './ValidatedEntryRow'

interface MemoryEntryRowProps {
  id: string
  value: number
  onValueChange: (mib: number) => void
  disabled?: boolean
  stacked?: boolean
  help?: string
}

const decimal = /^\d+(\.\d+)?$/

export function MemoryEntryRow({ id, value, onValueChange, disabled, stacked, help }: MemoryEntryRowProps) {
  const [text, setText] = useState(String(value / 1024))

  useEffect(() => {
    setText(String(value / 1024))
  }, [value])

  const mib = Number(text) * 1024
  const error =
    decimal.test(text) && Number.isSafeInteger(mib) && mib >= 64
      ? ''
      : 'At least 0.0625 GiB, in steps of 1 MiB'

  function change(next: string) {
    setText(next)
    const amount = Number(next) * 1024
    if (decimal.test(next) && Number.isSafeInteger(amount) && amount >= 64) onValueChange(amount)
  }

  return (
    <ValidatedEntryRow
      id={id}
      title="Memory (GiB)"
      stacked={stacked}
      required
      disabled={disabled}
      inputMode="decimal"
      value={text}
      error={error}
      help={help}
      onChange={(event) => change(event.target.value)}
    />
  )
}
