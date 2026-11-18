import { useState, type FormEvent } from 'react'
import { Button, EntryRow, PreferencesGroup } from 'maco-ui'
import { toast } from 'sonner'
import { changePassword, errorMessage } from '../api'
import { JobNotice } from './JobNotice'
import { StableWidth } from './StableWidth'

export function PasswordForm() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const tooShort = next.length > 0 && next.length < 8
  const mismatch = confirm.length > 0 && next !== confirm
  const valid = current.length > 0 && next.length >= 8 && next === confirm

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError('')
    try {
      await changePassword(current, next)
      toast.success('Password changed')
      setCurrent('')
      setNext('')
      setConfirm('')
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-3">
      <JobNotice error={error} />
      <PreferencesGroup title="Password">
        <EntryRow
          id="current-password"
          title="Current Password"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(event) => setCurrent(event.target.value)}
        />
        <EntryRow
          id="new-password"
          title="New Password"
          type="password"
          autoComplete="new-password"
          value={next}
          error={tooShort ? 'At least 8 characters' : ''}
          onChange={(event) => setNext(event.target.value)}
        />
        <EntryRow
          id="confirm-password"
          title="Confirm Password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          error={mismatch ? 'Passwords do not match' : ''}
          onChange={(event) => setConfirm(event.target.value)}
        />
      </PreferencesGroup>
      <div className="flex justify-end">
        <Button type="submit" disabled={!valid || busy}>
          <StableWidth value={busy ? 'Changing…' : 'Change Password'} values={['Change Password', 'Changing…']} />
        </Button>
      </div>
    </form>
  )
}
