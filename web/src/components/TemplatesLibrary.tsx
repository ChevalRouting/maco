import { useState } from 'react'
import { Pencil, Plus, Server, Trash2 } from 'lucide-react'
import { AlertDialog, Button, EmptyState, PreferencesGroup, Row, fmtBytes } from 'maco-ui'
import { listTemplates, deleteTemplate, errorMessage, type Template } from '../api'
import { useResource } from '../hooks/useResource'
import { useSession } from '../hooks/useSession'
import { ActionButton } from './ActionButton'
import { ResourceNotice } from './ResourceNotice'
import { TemplateSheet } from './TemplateSheet'
import { CreateVMSheet } from './CreateVMSheet'
import { JobNotice } from './JobNotice'
import { activatable } from '../hooks/useActivate'

function summary(template: Template): string {
  const spec = template.spec
  const provisioner = spec.guest_setup?.provisioner
  const parts = [
    spec.image || 'ISO',
    `${spec.cpus ?? 0} vCPU`,
    fmtBytes((spec.memory_mib ?? 0) * 1048576),
  ]
  if (provisioner && provisioner !== 'none') parts.push(provisioner)
  return parts.join(' · ')
}

interface TemplateRowProps {
  template: Template
  onEdit: () => void
  onCreateVM: () => void
  onDelete: () => void
}

function TemplateRow({ template, onEdit, onCreateVM, onDelete }: TemplateRowProps) {
  const { admin } = useSession()
  const activation = activatable(admin ? onEdit : undefined)

  return (
    <div {...activation}>
      <Row
        title={template.name}
        subtitle={
          <>
            {template.description && <span className="block">{template.description}</span>}
            <span className="block font-mono">{summary(template)}</span>
          </>
        }
      >
        {admin && (
          <>
            <ActionButton label={`Create VM from ${template.name}`} icon={Server} onClick={onCreateVM} />
            <ActionButton label={`Edit ${template.name}`} icon={Pencil} onClick={onEdit} />
            <ActionButton label={`Delete ${template.name}`} icon={Trash2} onClick={onDelete} />
          </>
        )}
      </Row>
    </div>
  )
}

export function TemplatesLibrary() {
  const { admin } = useSession()
  const templates = useResource(listTemplates, [], 'templates')
  const [editing, setEditing] = useState<Template | 'new' | null>(null)
  const [instantiating, setInstantiating] = useState<Template | null>(null)
  const [deleting, setDeleting] = useState<Template | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function remove() {
    if (!deleting) return
    setBusy(true)
    setError('')
    try {
      await deleteTemplate(deleting.id)
      setDeleting(null)
      templates.refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  function closeEditor() {
    setEditing(null)
    templates.refresh()
  }

  const createButton = admin && (
    <Button variant="outline" className="gap-2" onClick={() => setEditing('new')}>
      <Plus aria-hidden="true" className="h-4 w-4" />
      Create Template…
    </Button>
  )

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <ResourceNotice resource={templates} name="templates" />
      <JobNotice error={error} />
      {!templates.loading && !templates.error && templates.data.length === 0 ? (
        <EmptyState icon={<Server />} title="No Templates" action={createButton || undefined} />
      ) : (
        <PreferencesGroup title="Templates" header={createButton}>
          {templates.data.map((template) => (
            <TemplateRow
              key={template.id}
              template={template}
              onEdit={() => setEditing(template)}
              onCreateVM={() => setInstantiating(template)}
              onDelete={() => setDeleting(template)}
            />
          ))}
        </PreferencesGroup>
      )}
      {editing && <TemplateSheet template={editing === 'new' ? undefined : editing} onClose={closeEditor} />}
      {instantiating && <CreateVMSheet template={instantiating.id} onClose={() => setInstantiating(null)} />}
      <AlertDialog
        open={!!deleting}
        onCancel={() => setDeleting(null)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Delete ${deleting?.name ?? 'Template'}?`}
        description="Existing virtual machines are not affected"
        confirmLabel="Delete Template"
      />
    </div>
  )
}
