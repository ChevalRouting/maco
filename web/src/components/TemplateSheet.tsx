import { useState, type FormEvent } from 'react'
import { Button, EntryRow, PreferencesGroup, Sheet } from 'maco-ui'
import { createTemplate, errorMessage, updateTemplate, type Template } from '../api'
import { nameError } from '../ux'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { VMSpecFields } from './VMSpecFields'
import { useSpecResources } from './useSpecResources'
import { draftToSpec, specToDraft, specValid } from './vmSpec'
import { StableWidth } from './StableWidth'

interface TemplateSheetProps {
  template?: Template
  onClose: () => void
}

export function TemplateSheet({ template, onClose }: TemplateSheetProps) {
  const [name, setName] = useState(template?.name ?? '')
  const [description, setDescription] = useState(template?.description ?? '')
  const [draft, setDraft] = useState(() => specToDraft(template?.spec))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const { resources, host, loading, notices } = useSpecResources()
  const valid = !loading && !nameError(name) && specValid(draft, resources)

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || busy) return
    const body = { name, description, spec: draftToSpec(draft, resources, '') }
    setBusy(true)
    setError('')
    try {
      if (template) await updateTemplate(template.id, body)
      else await createTemplate(body)
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Sheet open onClose={() => !busy && onClose()} title={template ? `Edit ${template.name}` : 'Create Template'}>
      <form onSubmit={submit} className="space-y-5">
        <JobNotice error={error} />
        {notices.map((notice) => (
          <ResourceNotice key={notice.name} resource={notice.resource} name={notice.name} />
        ))}
        <fieldset disabled={busy} className="space-y-5">
          <PreferencesGroup>
            <ValidatedEntryRow
              id="template-name"
              title="Name"
              placeholder="ubuntu-server"
              required
              autoFocus
              value={name}
              error={name ? nameError(name) : ''}
              onChange={(event) => setName(event.target.value)}
            />
            <EntryRow
              id="template-description"
              title="Description"
              placeholder="Optional"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </PreferencesGroup>
          <VMSpecFields idPrefix="template" draft={draft} onChange={setDraft} resources={resources} host={host} disabled={busy} />
        </fieldset>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="suggested" disabled={!valid || busy}>
            <StableWidth value={busy ? 'Saving…' : template ? 'Save' : 'Create'} values={[template ? 'Save' : 'Create', 'Saving…']} />
          </Button>
        </div>
      </form>
    </Sheet>
  )
}
