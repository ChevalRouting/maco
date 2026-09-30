import { useState } from 'react'
import { Pencil, Trash2, Plus, Rocket } from 'lucide-react'
import { Button, PreferencesGroup, AlertDialog, fmtBytes } from 'cheval-ui'
import { listTemplates, deleteTemplate, errorMessage, type Template } from '../api'
import { useResource } from '../hooks/useResource'
import { useSession } from '../hooks/useSession'
import { ActionButton } from './ActionButton'
import { ResourceNotice } from './ResourceNotice'
import { TemplateEditor } from './TemplateEditor'
import { CreateVMDialog } from './CreateVMDialog'

function summary(template: Template): string {
  const spec = template.spec
  const provisioner = spec.guest_setup?.provisioner
  const guest = provisioner && provisioner !== 'none' ? provisioner : 'no guest config'
  const image = spec.image || 'installer'
  return `${image} · ${spec.cpus ?? '?'} vCPU · ${fmtBytes((spec.memory_mib ?? 0) * 1048576)} · ${guest}`
}

function TemplateRow({ template, onChange }: { template: Template; onChange: () => void }) {
  const { admin } = useSession()
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [editing, setEditing] = useState(false)
  const [instantiating, setInstantiating] = useState(false)

  async function remove() {
    setBusy(true)
    setError('')
    try {
      await deleteTemplate(template.id)
      setConfirm(false)
      onChange()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex items-center justify-between gap-4 p-4">
      <div className="min-w-0">
        <p className="break-words">{template.name}</p>
        {template.description && (
          <p className="text-sm text-muted-foreground break-words">{template.description}</p>
        )}
        <p className="text-sm text-muted-foreground">{summary(template)}</p>
        {error && <p className="text-sm text-destructive">{error}</p>}
      </div>

      <div className="flex items-center gap-2">
        {admin && (
          <Button size="sm" variant="suggested" onClick={() => setInstantiating(true)}>
            <Rocket className="mr-1 h-4 w-4" /> Create VM
          </Button>
        )}
        {admin && (
          <ActionButton label={`Edit ${template.name}`} icon={Pencil} onClick={() => setEditing(true)} />
        )}
        {admin && (
          <ActionButton
            label={`Delete ${template.name}`}
            icon={Trash2}
            busy={busy}
            onClick={() => setConfirm(true)}
          />
        )}
      </div>

      {editing && (
        <TemplateEditor
          template={template}
          onClose={() => {
            setEditing(false)
            onChange()
          }}
        />
      )}
      {instantiating && (
        <CreateVMDialog initial={{ spec: template.spec }} onClose={() => setInstantiating(false)} />
      )}
      {admin && (
        <AlertDialog
          open={confirm}
          onCancel={() => setConfirm(false)}
          onConfirm={remove}
          busy={busy}
          destructive
          title={`Delete ${template.name}?`}
          description="This removes the saved template. Existing VMs are unaffected."
        />
      )}
    </div>
  )
}

export function TemplatesLibrary() {
  const { admin } = useSession()
  const templates = useResource(listTemplates, [], 'templates')
  const [creating, setCreating] = useState(false)

  return (
    <div className="space-y-6">
      <ResourceNotice resource={templates} name="templates" />
      {admin && (
        <Button onClick={() => setCreating(true)}>
          <Plus className="mr-1 h-4 w-4" /> New Template
        </Button>
      )}
      <PreferencesGroup title="Saved Templates">
        {templates.data.map((template) => (
          <TemplateRow key={template.id} template={template} onChange={templates.refresh} />
        ))}
        {!templates.loading && !templates.data.length && !templates.error && (
          <p className="p-4 text-sm text-muted-foreground">
            No templates yet. {admin ? 'Create one above.' : 'An administrator can create one.'}
          </p>
        )}
      </PreferencesGroup>
      {creating && (
        <TemplateEditor
          onClose={() => {
            setCreating(false)
            templates.refresh()
          }}
        />
      )}
    </div>
  )
}
