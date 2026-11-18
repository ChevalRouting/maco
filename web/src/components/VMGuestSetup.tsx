import { useCallback, useEffect, useRef, useState } from 'react'
import { NoticeBanner, PreferencesGroup } from 'maco-ui'
import { getGuestSetup, updateGuestSetup, type GuestSetupView, type VMView, type Job } from '../api'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { FormActions } from './FormActions'
import { GuestSetup, type GuestSetupValue } from './GuestSetup'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'

function toValue(setup: GuestSetupView): GuestSetupValue {
  return {
    provisioner: setup.provisioner,
    files: (setup.files ?? []).map((file) => ({ name: file.name, content: file.content ?? '' })),
  }
}

interface VMGuestSetupProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMGuestSetup({ vm, run }: VMGuestSetupProps) {
  const manifest = vm.manifest
  const load = useCallback(() => getGuestSetup(manifest.id), [manifest.id])
  const setup = useResource<GuestSetupView | null>(load, null, 'vms')
  const action = useRowAction(manifest.id, manifest.name, 'vm', run)
  const [draft, setDraft] = useState<GuestSetupValue | null>(null)
  const [activeFile, setActiveFile] = useState('')
  const loaded = useRef(false)

  useEffect(() => {
    if (setup.data && !loaded.current) {
      loaded.current = true
      setDraft(toValue(setup.data))
    }
  }, [setup.data])

  const saved = setup.data ? JSON.stringify(toValue(setup.data)) : ''
  const dirty = !!draft && JSON.stringify(draft) !== saved
  const empty = !!draft && !draft.files.some((file) => file.content.trim())

  async function save() {
    if (!draft || action.busy || empty) return
    await action.execute('vm.guest-setup', () =>
      updateGuestSetup(manifest.id, { provisioner: draft.provisioner, mode: 'raw', files: draft.files }),
    )
  }

  function discard() {
    if (setup.data) setDraft(toValue(setup.data))
  }

  return (
    <div className="max-w-3xl space-y-6">
      <ResourceNotice resource={setup} name="guest setup" />
      <JobNotice error={action.error} />
      {vm.phase === 'running' && <NoticeBanner intent="info">Changes apply at the next boot</NoticeBanner>}
      {draft && setup.data && (
        <>
          <PreferencesGroup title="Configuration">
            <GuestSetup
              value={draft}
              onChange={setDraft}
              capabilities={setup.data.provisioning}
              disabled={action.busy}
              activeFile={activeFile}
              onActiveFileChange={setActiveFile}
            />
          </PreferencesGroup>
          <FormActions dirty={dirty} busy={action.busy} disabled={empty} onDiscard={discard} onSave={save} />
        </>
      )}
    </div>
  )
}
