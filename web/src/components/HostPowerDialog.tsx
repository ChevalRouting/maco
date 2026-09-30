import { useState } from 'react'
import { Dialog, Button, PreferencesGroup, SwitchRow, NoticeBanner } from 'cheval-ui'
import { powerOffHost, rebootHost } from '../api'
import { useJobAction } from '../hooks/useJobAction'
import { JobNotice } from './JobNotice'

export type HostPowerMode = 'poweroff' | 'reboot'

const copy: Record<HostPowerMode, { title: string; verb: string; description: string }> = {
  poweroff: {
    title: 'Power off host',
    verb: 'Power off',
    description: 'Running virtual machines are shut down cleanly before the hypervisor powers off.',
  },
  reboot: {
    title: 'Reboot host',
    verb: 'Reboot',
    description: 'Running virtual machines are shut down cleanly before the hypervisor reboots.',
  },
}

export function HostPowerDialog({ mode, onClose }: { mode: HostPowerMode; onClose: () => void }) {
  const [force, setForce] = useState(false)
  const action = useJobAction()
  const text = copy[mode]

  function close() {
    if (action.busy) return
    onClose()
  }

  async function submit() {
    const result = await action.run(() => (mode === 'poweroff' ? powerOffHost(force) : rebootHost(force)))
    if (result) onClose()
  }

  return (
    <Dialog
      open
      onClose={close}
      title={text.title}
      description={text.description}
      footer={
        <>
          <Button variant="outline" onClick={close} disabled={action.busy}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={submit} disabled={action.busy}>
            {action.busy ? 'Submitting…' : text.verb}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <JobNotice error={action.error} />
        <PreferencesGroup title="Options">
          <SwitchRow
            title="Force"
            subtitle="Skip the graceful guest shutdown and power the host immediately."
            checked={force}
            onCheckedChange={setForce}
            disabled={action.busy}
          />
        </PreferencesGroup>
        {force && (
          <NoticeBanner intent="warning">
            Forcing bypasses virtual machine shutdown. Unsaved guest data may be lost.
          </NoticeBanner>
        )}
      </div>
    </Dialog>
  )
}
