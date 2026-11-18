import { Button } from 'maco-ui'
import { StableWidth } from './StableWidth'

interface FormActionsProps {
  dirty: boolean
  busy: boolean
  disabled?: boolean
  onDiscard: () => void
  onSave?: () => void
}

export function FormActions({ dirty, busy, disabled, onDiscard, onSave }: FormActionsProps) {
  return (
    <div className="flex justify-end gap-2">
      <Button type="button" variant="outline" disabled={!dirty || busy} onClick={onDiscard}>
        Discard Changes
      </Button>
      <Button type={onSave ? 'button' : 'submit'} variant="suggested" disabled={!dirty || busy || disabled} onClick={onSave}>
        <StableWidth value={busy ? 'Saving…' : 'Save'} values={['Save', 'Saving…']} />
      </Button>
    </div>
  )
}
