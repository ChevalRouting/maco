import { useCallback, useEffect, useRef, useState } from 'react'
import { Button, NoticeBanner } from 'cheval-ui'
import { getGuestSetup, updateGuestSetup, type GuestSetupView, type VMView, type Job } from '../api'
import { useResource } from '../hooks/useResource'
import { useRowAction } from '../hooks/useRowAction'
import { GuestSetup, type GuestSetupValue } from './GuestSetup'
import { TemplateVariables } from './TemplateVariables'
import { ResourceNotice } from './ResourceNotice'

function toValue(setup: GuestSetupView): GuestSetupValue {
  return {
    provisioner: setup.provisioner,
    files: (setup.files ?? []).map((file) => ({ name: file.name, content: file.content ?? '' })),
  }
}

export function VMGuestSetup({ vm, run }: { vm: VMView; run: (action: () => Promise<Job>) => Promise<Job | null> }) {
  const manifest = vm.manifest
  const load = useCallback(() => getGuestSetup(manifest.id), [manifest.id])
  const setup = useResource<GuestSetupView | null>(load, null, 'vms')
  const action = useRowAction(manifest.id, manifest.name, 'vm', run)
  const blocked = action.busy

  const [draft, setDraft] = useState<GuestSetupValue | null>(null)
  const [activeFile, setActiveFile] = useState('')
  const loaded = useRef(false)
  useEffect(() => {
    if (setup.data && !loaded.current) {
      loaded.current = true
      setDraft(toValue(setup.data))
    }
  }, [setup.data])

  const rawMissing = !!draft && !draft.files.some((file) => file.content.trim())

  async function save() {
    if (!draft || blocked || rawMissing) return
    await action.execute('vm.guest-setup', () =>
      updateGuestSetup(manifest.id, {
        provisioner: draft.provisioner,
        mode: 'raw',
        files: draft.files,
      }),
    )
  }

  return (
    <div className="max-w-2xl space-y-4">
      <ResourceNotice resource={setup} name="guest setup" />
      {vm.phase === 'running' && (
        <NoticeBanner intent="info">
          Guest setup is applied during boot. You can edit and save it while the VM is running, but
          the guest must be rebooted for the changes to take effect.
        </NoticeBanner>
      )}
      {draft && setup.data && (
        <>
          <TemplateVariables />
          <GuestSetup
            value={draft}
            onChange={setDraft}
            capabilities={setup.data.provisioning}
            disabled={blocked}
            activeFile={activeFile}
            onActiveFileChange={setActiveFile}
          />
          <div className="flex justify-end">
            <Button variant="suggested" onClick={save} disabled={blocked || rawMissing}>
              {action.busy ? 'Saving…' : 'Save guest setup'}
            </Button>
          </div>
        </>
      )}
    </div>
  )
}
