import { useState } from 'react'
import { Dialog, Button, PreferencesGroup, SwitchRow } from 'maco-ui'
import { powerOffHost, rebootHost } from '../api'
import { useJobAction } from '../hooks/useJobAction'
import { JobNotice } from './JobNotice'
import { StableWidth } from './StableWidth'

export type HostPowerMode = 'poweroff' | 'reboot'

interface PowerText {
  title: string
  verb: string
}

const text: Record<HostPowerMode, PowerText> = {
  poweroff: { title: 'Power Off Host', verb: 'Power Off' },
  reboot: { title: 'Reboot Host', verb: 'Reboot' },
}

interface HostPowerDialogProps {
  mode: HostPowerMode
  onClose: () => void
}

export function HostPowerDialog({ mode, onClose }: HostPowerDialogProps) {
  const [force, setForce] = useState(false)
  const action = useJobAction()

  function close() {
    if (!action.busy) onClose()
  }

  async function submit() {
    const result = await action.run(() => (mode === 'poweroff' ? powerOffHost(force) : rebootHost(force)))
    if (result) onClose()
  }

  return (
    <Dialog
      open
      onClose={close}
      title={text[mode].title}
      description="Running virtual machines are shut down first"
      footer={
        <>
          <Button variant="outline" onClick={close} disabled={action.busy}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={submit} disabled={action.busy}>
            <StableWidth value={action.busy ? 'Submitting…' : text[mode].verb} values={[text[mode].verb, 'Submitting…']} />
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <JobNotice error={action.error} />
        <PreferencesGroup>
          <SwitchRow
            title="Skip Guest Shutdown"
            subtitle="Unsaved guest data is lost"
            checked={force}
            onCheckedChange={setForce}
            disabled={action.busy}
          />
        </PreferencesGroup>
      </div>
    </Dialog>
  )
}
