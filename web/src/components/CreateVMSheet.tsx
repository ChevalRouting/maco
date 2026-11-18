import { useState, type FormEvent } from 'react'
import { Button, PreferencesGroup, Sheet } from 'maco-ui'
import { createVM, listTemplates, type CreateVMParams } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { nameError } from '../ux'
import { ChoiceRow } from './ChoiceRow'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { ValidatedEntryRow } from './ValidatedEntryRow'
import { VMSpecFields } from './VMSpecFields'
import { useSpecResources } from './useSpecResources'
import { draftToSpec, specToDraft, specValid } from './vmSpec'
import { StableWidth } from './StableWidth'

interface CreateVMSheetProps {
  onClose: () => void
  template?: string
}

interface CreateVMFormProps {
  spec?: CreateVMParams
  templateRow: React.ReactNode
  onClose: () => void
}

function CreateVMForm({ spec, templateRow, onClose }: CreateVMFormProps) {
  const [name, setName] = useState('')
  const [draft, setDraft] = useState(() => specToDraft(spec))
  const { resources, host, loading, notices } = useSpecResources()
  const action = useJobAction()
  const valid = !loading && !nameError(name) && specValid(draft, resources)

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!valid || action.busy) return
    const job = await action.run(() => createVM(draftToSpec(draft, resources, name)))
    if (job) onClose()
  }

  return (
    <form onSubmit={submit} className="space-y-5">
      <JobNotice error={action.error} />
      {notices.map((notice) => (
        <ResourceNotice key={notice.name} resource={notice.resource} name={notice.name} />
      ))}
      <fieldset disabled={action.busy} className="space-y-5">
        <PreferencesGroup>
          {templateRow}
          <ValidatedEntryRow
            id="vm-name"
            title="Name"
            placeholder="web-01"
            required
            autoFocus
            value={name}
            error={name ? nameError(name) : ''}
            onChange={(event) => setName(event.target.value)}
          />
        </PreferencesGroup>
        <VMSpecFields idPrefix="vm" draft={draft} onChange={setDraft} resources={resources} host={host} disabled={action.busy} />
      </fieldset>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" disabled={action.busy} onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" variant="suggested" disabled={!valid || action.busy}>
          <StableWidth value={action.busy ? 'Creating…' : 'Create'} values={['Create', 'Creating…']} />
        </Button>
      </div>
    </form>
  )
}

export function CreateVMSheet({ onClose, template = '' }: CreateVMSheetProps) {
  const [templateId, setTemplateId] = useState(template)
  const templates = useResource(listTemplates, [], 'templates')
  const selected = templates.data.find((item) => item.id === templateId)

  const templateRow = templates.data.length > 0 && (
    <ChoiceRow
      title="Template"
      value={templateId}
      choices={[{ value: '', label: 'None' }, ...templates.data.map((item) => ({ value: item.id, label: item.name }))]}
      help={selected?.description || undefined}
      onChange={setTemplateId}
    />
  )

  return (
    <Sheet open onClose={onClose} title="Create Virtual Machine">
      {templates.loading ? null : (
        <CreateVMForm key={selected?.id ?? ''} spec={selected?.spec} templateRow={templateRow} onClose={onClose} />
      )}
    </Sheet>
  )
}
