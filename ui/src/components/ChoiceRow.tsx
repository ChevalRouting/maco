import { useId } from 'react'
import { ComboRow } from '@/components/Preferences'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select'

export interface Choice {
  value: string
  label: string
  disabled?: boolean
}

export interface ChoiceRowProps {
  title: string
  value: string
  choices: Choice[]
  onChange: (value: string) => void
  disabled?: boolean
  stacked?: boolean
  help?: string
}

export function ChoiceRow({
  title,
  value,
  choices,
  onChange,
  disabled,
  stacked,
  help,
}: ChoiceRowProps) {
  const id = useId()
  let emptyValue = `${id}-empty`
  while (choices.some((choice) => choice.value === emptyValue)) {
    emptyValue += '-'
  }
  const hasNone = choices.some((item) => item.value === '')

  const control = (
    <Select
      value={hasNone && value === '' ? emptyValue : value}
      onValueChange={(next) => onChange(next === emptyValue ? '' : next)}
      disabled={disabled || !choices.length}
    >
      <SelectTrigger
        id={id}
        aria-label={title}
        aria-describedby={help ? `${id}-help` : undefined}
        className="min-w-0 h-auto min-h-8 [&>span]:min-w-0 [&>span]:whitespace-normal [&>span]:line-clamp-none"
      >
        <SelectValue
          placeholder={
            choices.length ? `Choose ${title.toLowerCase()}` : `No ${title.toLowerCase()} available`
          }
        />
      </SelectTrigger>
      <SelectContent
        className="z-[60] max-w-[calc(100vw-2rem)]"
      >
        {choices.map((item) => (
          <SelectItem
            key={item.value || emptyValue}
            value={item.value || emptyValue}
            disabled={item.disabled}
            className="whitespace-normal break-words"
          >
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  if (stacked) {
    return (
      <div
        className="flex min-w-0 flex-col gap-1.5 px-4 py-2"
      >
        <label
          htmlFor={id}
          className="text-xs font-medium text-muted-foreground"
        >
          {title}
        </label>
        {control}
        {help && (
          <p
            id={`${id}-help`}
            className="text-xs text-muted-foreground"
          >
            {help}
          </p>
        )}
      </div>
    )
  }

  return (
    <ComboRow
      title={title}
      subtitle={help && (
        <span
          id={`${id}-help`}
        >
          {help}
        </span>
      )}
    >
      {control}
    </ComboRow>
  )
}
