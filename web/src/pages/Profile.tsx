import { useState, type FormEvent } from 'react'
import { PageHeader, PreferencesGroup, EntryRow, Button, NoticeBanner } from 'cheval-ui'
import { changePassword, errorMessage } from '../api'
import { useSession } from '../hooks/useSession'
import { JobNotice } from '../components/JobNotice'
import { SSHKeys } from '../components/SSHKeys'

export default function Profile() {
  const session = useSession()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  const tooShort = next.length > 0 && next.length < 8
  const mismatch = confirm.length > 0 && next !== confirm
  const valid = current.length > 0 && next.length >= 8 && next === confirm

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError('')
    setSaved(false)
    try {
      await changePassword(current, next)
      setSaved(true)
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
    <div className="max-w-xl space-y-6">
      <PageHeader title="Profile" description="Your maco account." />

      <PreferencesGroup title="Account">
        <div className="px-4 py-3 text-sm">
          <p className="break-words font-medium">{session.user?.username || 'Unknown account'}</p>
          <p className="text-muted-foreground">
            {session.user ? (session.admin ? 'Administrator' : 'Read-only Access') : ''}
          </p>
        </div>
      </PreferencesGroup>

      <form onSubmit={submit} className="space-y-4">
        <JobNotice error={error} />
        {saved && <NoticeBanner intent="success">Your password was updated.</NoticeBanner>}

        <PreferencesGroup title="Change Password">
          <EntryRow
            id="current-password"
            title="Current password"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(event) => {
              setCurrent(event.target.value)
              setSaved(false)
            }}
          />
          <EntryRow
            id="new-password"
            title="New password"
            type="password"
            autoComplete="new-password"
            value={next}
            error={tooShort ? 'Use at least 8 characters.' : ''}
            help="At least 8 characters."
            onChange={(event) => {
              setNext(event.target.value)
              setSaved(false)
            }}
          />
          <EntryRow
            id="confirm-password"
            title="Confirm new password"
            type="password"
            autoComplete="new-password"
            value={confirm}
            error={mismatch ? 'Passwords do not match.' : ''}
            onChange={(event) => {
              setConfirm(event.target.value)
              setSaved(false)
            }}
          />
        </PreferencesGroup>

        <Button type="submit" variant="suggested" disabled={!valid || busy}>
          {busy ? 'Updating…' : 'Update Password'}
        </Button>
      </form>

      <SSHKeys />
    </div>
  )
}
