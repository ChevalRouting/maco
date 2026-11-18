import { Dialog, PreferencesGroup, Row, Spinner } from 'maco-ui'
import { getHost } from '../../api'
import { useResource } from '../../hooks/useResource'
import { memoryLabel } from '../../ux'
import { Logo } from '../../Logo'

interface AboutDialogProps {
  open: boolean
  onClose: () => void
}

function HostDetails() {
  const host = useResource(getHost, null, 'host')

  if (!host.data) return host.loading ? <Spinner /> : null

  return (
    <PreferencesGroup>
      <Row title="Host CPUs">
        <span className="text-sm">{host.data.cpus}</span>
      </Row>
      <Row title="Host Memory">
        <span className="text-sm">{memoryLabel(host.data.memory_bytes / 1048576)}</span>
      </Row>
    </PreferencesGroup>
  )
}

export function AboutDialog({ open, onClose }: AboutDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} title="About maco" className="max-w-sm">
      <div className="space-y-5">
        <div className="flex flex-col items-center gap-2 text-center">
          <Logo className="h-16 w-16" />
          <p className="text-base font-semibold">maco</p>
          <p className="text-sm text-muted-foreground">Virtual machine manager</p>
        </div>
        {open && <HostDetails />}
      </div>
    </Dialog>
  )
}
