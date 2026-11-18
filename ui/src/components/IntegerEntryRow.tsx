import { forwardRef, useEffect, useRef, useState, type ComponentProps } from 'react'
import { EntryRow } from '@/components/Preferences'

export interface IntegerEntryRowProps extends Omit<
  ComponentProps<typeof EntryRow>,
  'value' | 'defaultValue' | 'onChange' | 'type' | 'min' | 'max' | 'step' | 'error'
> {
  value: number
  min: number
  max?: number
  onValueChange: (value: number) => void
}

function integerError(value: string, min: number, max: number) {
  const digits = min < 0 ? /^-?[0-9]+$/ : /^[0-9]+$/
  if (!digits.test(value) || !Number.isSafeInteger(Number(value))) {
    return 'Enter a whole number using digits only.'
  }
  if (Number(value) < min) {
    return `Enter a value of at least ${min}.`
  }
  if (Number(value) > max) {
    return `Enter a value no greater than ${max}.`
  }
  return ''
}

export const IntegerEntryRow = forwardRef<HTMLInputElement, IntegerEntryRowProps>(
  ({ value, min, max = Number.MAX_SAFE_INTEGER, onValueChange, ...props }, ref) => {
    const [text, setText] = useState(String(value))
    const lastEmitted = useRef(value)
    const error = integerError(text, min, max)

    useEffect(() => {
      if (value !== lastEmitted.current) {
        setText(String(value))
      }
      lastEmitted.current = value
    }, [value])

    function change(event: React.ChangeEvent<HTMLInputElement>) {
      const next = event.target.value
      const message = integerError(next, min, max)
      setText(next)
      event.target.setCustomValidity(message)
      if (!message) {
        lastEmitted.current = Number(next)
        onValueChange(Number(next))
      }
    }

    return (
      <EntryRow
        {...props}
        ref={ref}
        type="text"
        inputMode={min < 0 ? 'text' : 'numeric'}
        value={text}
        onChange={change}
        error={error}
      />
    )
  },
)
IntegerEntryRow.displayName = 'IntegerEntryRow'
