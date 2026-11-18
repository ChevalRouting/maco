import { useState, type FormEvent } from 'react'
import { AlertDialog, Button, EntryRow, PreferencesGroup, Row, Sheet, Textarea } from 'maco-ui'
import { Plus, Trash2 } from 'lucide-react'
import { addSSHKey, listSSHKeys, deleteSSHKey, errorMessage, type SSHKey } from '../api'
import { useResource } from '../hooks/useResource'
import { ActionButton } from './ActionButton'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'
import { StableWidth } from './StableWidth'

const sshKeyPattern = /^(ssh-|ecdsa-|sk-)[^\s]+\s+[A-Za-z0-9+/]+={0,3}(\s.*)?$/

interface AddKeySheetProps {
  onClose: () => void
  onAdded: () => void
}

function AddKeySheet({ onClose, onAdded }: AddKeySheetProps) {
  const [name, setName] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const trimmed = publicKey.trim()
  const keyError = trimmed && !sshKeyPattern.test(trimmed) ? 'Not an SSH public key' : ''
  const valid = !!name.trim() && !!trimmed && !keyError

  async function add(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError('')
    try {
      await addSSHKey(name.trim(), trimmed)
      onAdded()
    } catch (e) {
      setError(errorMessage(e))
      setBusy(false)
    }
  }

  return (
    <Sheet open onClose={() => !busy && onClose()} title="Add SSH Key">
      <form onSubmit={add} className="space-y-5">
        <JobNotice error={error} />
        <PreferencesGroup>
          <EntryRow id="ssh-key-name" title="Name" placeholder="laptop" autoFocus value={name} onChange={(event) => setName(event.target.value)} />
          <div className="flex min-w-0 flex-col gap-1.5 px-4 py-3">
            <label htmlFor="ssh-key-value" className="text-sm font-medium">Public Key</label>
            <Textarea
              id="ssh-key-value"
              value={publicKey}
              spellCheck={false}
              rows={4}
              placeholder="ssh-ed25519 AAAA… user@host"
              className="font-mono text-xs"
              aria-invalid={keyError ? true : undefined}
              onChange={(event) => setPublicKey(event.target.value)}
            />
            {keyError && <p className="text-xs text-danger">{keyError}</p>}
          </div>
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={!valid || busy}>
            <StableWidth value={busy ? 'Adding…' : 'Add'} values={['Add', 'Adding…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}

export function SSHKeys() {
  const keys = useResource(listSSHKeys, [], 'ssh-keys')
  const [adding, setAdding] = useState(false)
  const [deleting, setDeleting] = useState<SSHKey | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function remove() {
    if (!deleting || busy) return
    setBusy(true)
    setError('')
    try {
      await deleteSSHKey(deleting.id)
      setDeleting(null)
      keys.refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  function added() {
    setAdding(false)
    keys.refresh()
  }

  return (
    <div className="space-y-3">
      <JobNotice error={error} />
      <ResourceNotice resource={keys} name="SSH keys" />
      <PreferencesGroup
        title="SSH Keys"
        description="Available to guest setup as [[ .SSHKeys ]]"
        header={
          <Button variant="outline" className="gap-2" onClick={() => setAdding(true)}>
            <Plus aria-hidden="true" className="h-4 w-4" />
            Add SSH Key…
          </Button>
        }
      >
        {!keys.loading && !keys.error && keys.data.length === 0 && <Row subtitle="None" />}
        {keys.data.map((key) => (
          <Row key={key.id} title={key.name} subtitle={<span className="break-all font-mono">{key.public_key}</span>}>
            <ActionButton label={`Delete ${key.name}`} icon={Trash2} disabled={busy} onClick={() => setDeleting(key)} />
          </Row>
        ))}
      </PreferencesGroup>
      {adding && <AddKeySheet onClose={() => setAdding(false)} onAdded={added} />}
      <AlertDialog
        open={!!deleting}
        onCancel={() => setDeleting(null)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Delete ${deleting?.name || 'SSH Key'}?`}
        description="New virtual machines no longer receive it; existing ones keep it"
        confirmLabel="Delete Key"
      />
    </div>
  )
}
