import { useState, type FormEvent } from 'react'
import { AlertDialog, Button, EntryRow, PreferencesGroup } from 'cheval-ui'
import { Trash2 } from 'lucide-react'
import { addSSHKey, listSSHKeys, deleteSSHKey, errorMessage, type SSHKey } from '../api'
import { useResource } from '../hooks/useResource'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'

const sshKeyPattern = /^(ssh-|ecdsa-|sk-)[^\s]+\s+[A-Za-z0-9+/]+={0,3}(\s.*)?$/

function formatDate(seconds?: number | null) {
  if (!seconds) return 'Unknown'
  return new Date(seconds * 1000).toLocaleString()
}

export function SSHKeys() {
  const keys = useResource(listSSHKeys, [], 'ssh-keys')
  const [name, setName] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState<SSHKey | null>(null)

  const trimmed = publicKey.trim()
  const keyError = trimmed && !sshKeyPattern.test(trimmed) ? 'Enter a complete SSH public key.' : ''
  const valid = !!name.trim() && !!trimmed && !keyError

  async function create(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    setBusy(true)
    setError('')
    try {
      await addSSHKey(name.trim(), trimmed)
      setName('')
      setPublicKey('')
      keys.refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

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

  return (
    <div className="max-w-3xl space-y-6">
      <JobNotice error={error} />

      <form onSubmit={create}>
        <PreferencesGroup
          title="Add SSH Key"
          description="Keys stored here populate [[ .SSHKeys ]] in guest configurations when you create a VM."
        >
          <EntryRow
            id="ssh-key-name"
            title="Name"
            placeholder="For example, laptop"
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
          <div className="flex min-w-0 flex-col gap-1 px-4 py-2">
            <label htmlFor="ssh-key-value" className="text-xs font-medium text-muted-foreground">
              Public key
            </label>
            <textarea
              id="ssh-key-value"
              value={publicKey}
              spellCheck={false}
              rows={3}
              placeholder="ssh-ed25519 AAAA… user@host"
              className="w-full rounded-lg border bg-card p-3 font-mono text-xs"
              onChange={(event) => setPublicKey(event.target.value)}
            />
            {keyError && <p className="text-xs text-destructive">{keyError}</p>}
            <p className="text-xs text-muted-foreground">
              Paste a public key, for example ssh-ed25519 AAAA…; never a private key.
            </p>
          </div>
          <div className="p-4">
            <Button type="submit" variant="suggested" disabled={!valid || busy}>
              {busy ? 'Adding…' : 'Add SSH Key'}
            </Button>
          </div>
        </PreferencesGroup>
      </form>

      <ResourceNotice resource={keys} name="SSH keys" />

      <PreferencesGroup title="Your SSH Keys">
        {keys.data.map((key) => (
          <div key={key.id} className="flex flex-wrap items-center gap-3 p-4">
            <div className="min-w-0 flex-1">
              <p className="break-words font-medium">{key.name}</p>
              <p className="break-all font-mono text-xs text-muted-foreground">{key.public_key}</p>
              <p className="text-sm text-muted-foreground">Added {formatDate(key.created_at)}</p>
            </div>
            <Button size="sm" variant="outline" disabled={busy} onClick={() => setDeleting(key)}>
              <Trash2 aria-hidden="true" className="mr-2 h-4 w-4" />
              Delete…
            </Button>
          </div>
        ))}
        {!keys.loading && !keys.error && keys.data.length === 0 && (
          <p className="p-4 text-sm text-muted-foreground">
            No SSH keys yet. Add one above to reference it as [[ .SSHKeys ]] in guest configurations.
          </p>
        )}
      </PreferencesGroup>

      <AlertDialog
        open={!!deleting}
        onCancel={() => setDeleting(null)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Delete ${deleting?.name || 'SSH key'}?`}
        description="New VMs will no longer receive this key. Existing VMs keep the keys they were created with."
        confirmLabel="Delete Key"
      />
    </div>
  )
}
