import { useState, type FormEvent } from 'react'
import { AlertDialog, Button, CopyButton, Dialog, EmptyState, EntryRow, PreferencesGroup, Row, Sheet } from 'maco-ui'
import { KeyRound, Plus, Trash2 } from 'lucide-react'
import { createAPIKey, listAPIKeys, revokeAPIKey, errorMessage, type APIKey } from '../api'
import { useResource } from '../hooks/useResource'
import { ActionButton } from './ActionButton'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'
import { StableWidth } from './StableWidth'

function formatDate(seconds?: number | null) {
  if (!seconds) return 'never'
  return new Date(seconds * 1000).toLocaleString()
}

interface CreatedKey {
  key: APIKey
  token: string
}

interface CreateKeySheetProps {
  onClose: () => void
  onCreated: (created: CreatedKey) => void
}

function CreateKeySheet({ onClose, onCreated }: CreateKeySheetProps) {
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function create(event: FormEvent) {
    event.preventDefault()
    if (!name.trim() || busy) return
    setBusy(true)
    setError('')
    try {
      const result = await createAPIKey(name.trim())
      onCreated({ key: result.api_key, token: result.token })
    } catch (e) {
      setError(errorMessage(e))
      setBusy(false)
    }
  }

  return (
    <Sheet open onClose={() => !busy && onClose()} title="Create API Key">
      <form onSubmit={create} className="space-y-5">
        <JobNotice error={error} />
        <PreferencesGroup>
          <EntryRow
            id="api-key-name"
            title="Name"
            placeholder="ci-pipeline"
            autoFocus
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </PreferencesGroup>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={!name.trim() || busy}>
            <StableWidth value={busy ? 'Creating…' : 'Create'} values={['Create', 'Creating…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}

interface CreatedKeyDialogProps {
  created: CreatedKey
  onClose: () => void
}

function CreatedKeyDialog({ created, onClose }: CreatedKeyDialogProps) {
  return (
    <Dialog
      open
      onClose={onClose}
      title={`API Key ${created.key.name}`}
      description="Shown only once; it carries your account’s permissions"
      footer={<Button onClick={onClose}>Close</Button>}
    >
      <div className="flex items-start gap-2">
        <code className="min-w-0 flex-1 break-all rounded-md bg-muted px-3 py-2 font-mono text-xs">{created.token}</code>
        <CopyButton text={created.token} label="Copy API Key" />
      </div>
    </Dialog>
  )
}

export function ApiKeys() {
  const keys = useResource(listAPIKeys, [], 'api-keys')
  const [creating, setCreating] = useState(false)
  const [created, setCreated] = useState<CreatedKey | null>(null)
  const [revoking, setRevoking] = useState<APIKey | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function revoke() {
    if (!revoking || busy) return
    setBusy(true)
    setError('')
    try {
      await revokeAPIKey(revoking.id)
      setRevoking(null)
      keys.refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  function showCreated(next: CreatedKey) {
    setCreating(false)
    setCreated(next)
    keys.refresh()
  }

  const createButton = (
    <Button variant="outline" className="gap-2" onClick={() => setCreating(true)}>
      <Plus aria-hidden="true" className="h-4 w-4" />
      Create API Key…
    </Button>
  )

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <JobNotice error={error} />
      <ResourceNotice resource={keys} name="API keys" />
      {!keys.loading && !keys.error && keys.data.length === 0 ? (
        <EmptyState icon={<KeyRound />} title="No API Keys" action={createButton} />
      ) : (
        <PreferencesGroup title="API Keys" header={createButton}>
          {keys.data.map((key) => (
            <Row
              key={key.id}
              title={key.name}
              subtitle={
                <>
                  <span className="font-mono">{key.prefix}…</span> · created {formatDate(key.created_at)} · last used {formatDate(key.last_used_at)}
                </>
              }
            >
              <ActionButton label={`Revoke ${key.name}`} icon={Trash2} disabled={busy} onClick={() => setRevoking(key)} />
            </Row>
          ))}
        </PreferencesGroup>
      )}
      {creating && <CreateKeySheet onClose={() => setCreating(false)} onCreated={showCreated} />}
      {created && <CreatedKeyDialog created={created} onClose={() => setCreated(null)} />}
      <AlertDialog
        open={!!revoking}
        onCancel={() => setRevoking(null)}
        onConfirm={revoke}
        busy={busy}
        destructive
        title={`Revoke ${revoking?.name || 'API Key'}?`}
        description="Clients using this key lose access immediately"
        confirmLabel="Revoke Key"
      />
    </div>
  )
}
