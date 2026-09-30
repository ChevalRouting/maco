import { useState } from 'react'
import { Dialog, Button } from 'cheval-ui'
import { FilePlus2, LayoutTemplate } from 'lucide-react'
import { listTemplates } from '../api'
import { useResource } from '../hooks/useResource'
import { ChoiceRow } from './ChoiceRow'
import { ResourceNotice } from './ResourceNotice'
import { CreateVMDialog } from './CreateVMDialog'

type Mode = 'scratch' | 'template'

interface OptionCardProps {
  title: string
  description: string
  icon: typeof FilePlus2
  selected: boolean
  onSelect: () => void
}

function OptionCard({ title, description, icon: Icon, selected, onSelect }: OptionCardProps) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onSelect}
      className={`flex flex-col items-start gap-2 rounded-xl border p-4 text-left transition-colors ${
        selected ? 'border-primary ring-2 ring-ring' : 'hover:bg-accent'
      }`}
    >
      <Icon className="h-5 w-5 shrink-0" />
      <span className="font-semibold">{title}</span>
      <span className="text-sm text-muted-foreground">{description}</span>
    </button>
  )
}

export function CreateVMChooser({ onClose }: { onClose: () => void }) {
  const [mode, setMode] = useState<Mode>('scratch')
  const [templateId, setTemplateId] = useState('')
  const [launched, setLaunched] = useState(false)
  const templates = useResource(listTemplates, [], 'templates')

  const selected = templates.data.find((template) => template.id === templateId)

  if (launched) {
    return (
      <CreateVMDialog
        initial={mode === 'template' && selected ? { spec: selected.spec } : undefined}
        onClose={onClose}
      />
    )
  }

  const canContinue = mode === 'scratch' || !!selected

  return (
    <Dialog
      open
      onClose={onClose}
      title="Create Virtual Machine"
      description="Start from a blank configuration, or pre-fill from a saved template."
      className="max-w-lg"
      footer={
        <>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="suggested" disabled={!canContinue} onClick={() => setLaunched(true)}>
            Continue
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          <OptionCard
            title="Create From Scratch"
            description="Configure a new VM from defaults."
            icon={FilePlus2}
            selected={mode === 'scratch'}
            onSelect={() => setMode('scratch')}
          />
          <OptionCard
            title="Create from Template"
            description="Pre-fill from a saved template."
            icon={LayoutTemplate}
            selected={mode === 'template'}
            onSelect={() => setMode('template')}
          />
        </div>
        {mode === 'template' && (
          <>
            <ResourceNotice resource={templates} name="templates" />
            {!templates.loading && !templates.data.length && !templates.error ? (
              <p className="px-4 text-sm text-muted-foreground">
                No templates yet. Create one from the Templates page first.
              </p>
            ) : (
              <ChoiceRow
                stacked
                title="Template"
                value={templateId}
                choices={templates.data.map((template) => ({
                  value: template.id,
                  label: template.name,
                }))}
                onChange={setTemplateId}
                help={selected?.description}
              />
            )}
          </>
        )}
      </div>
    </Dialog>
  )
}
