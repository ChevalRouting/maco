import { useCallback } from 'react'
import { CopyButton, PreferencesGroup, ReloadButton, Row } from 'maco-ui'
import { getGuestAgent, type GuestAgent as GuestAgentData } from '../api'
import { useResource } from '../hooks/useResource'
import { formatBytes } from '../ux'
import { ResourceNotice } from './ResourceNotice'

interface GuestAgentProps {
  id: string
}

export function GuestAgent({ id }: GuestAgentProps) {
  const load = useCallback(() => getGuestAgent(id), [id])
  const agent = useResource<GuestAgentData | null>(load, null, `guest-agent:${id}`)
  const data = agent.data
  const os = data?.os
  const interfaces = (data?.interfaces || []).filter((nic) => nic.name !== 'lo' && (nic.ip_addresses?.length || 0) > 0)
  const filesystems = (data?.filesystems || []).filter((fs) => fs.total_bytes > 0)

  return (
    <div className="space-y-6">
      <ResourceNotice resource={agent} name="guest information" />
      <PreferencesGroup
        title="Guest"
        description={data?.available && data.version ? `qemu-guest-agent ${data.version}` : undefined}
        header={<ReloadButton onClick={agent.refresh} loading={agent.loading} />}
      >
        {!agent.loading && data && !data.available && <Row title="Agent" subtitle="Not running in the guest" />}
        {data?.hostname && (
          <Row title="Hostname">
            <span className="font-mono text-sm">{data.hostname}</span>
          </Row>
        )}
        {os?.pretty_name && (
          <Row title="Operating System">
            <span className="text-sm">{os.pretty_name}</span>
          </Row>
        )}
        {os?.kernel_release && (
          <Row title="Kernel">
            <span className="font-mono text-sm">{`${os.kernel_release}${os.machine ? ` ${os.machine}` : ''}`}</span>
          </Row>
        )}
      </PreferencesGroup>

      {interfaces.length > 0 && (
        <PreferencesGroup title="Guest Addresses">
          {interfaces.map((nic) => (
            <Row key={nic.name} title={nic.name}>
              <div className="flex flex-col items-end gap-1">
                {(nic.ip_addresses || []).map((addr) => (
                  <span key={addr.address} className="flex items-center gap-1">
                    <span className="break-all font-mono text-sm">{`${addr.address}/${addr.prefix}`}</span>
                    <CopyButton text={addr.address} label={`Copy ${addr.address}`} />
                  </span>
                ))}
              </div>
            </Row>
          ))}
        </PreferencesGroup>
      )}

      {filesystems.length > 0 && (
        <PreferencesGroup title="Guest Filesystems">
          {filesystems.map((fs) => (
            <Row key={fs.mountpoint} title={fs.mountpoint} subtitle={fs.type}>
              <span className="text-sm tabular-nums">{`${formatBytes(fs.used_bytes)} of ${formatBytes(fs.total_bytes)}`}</span>
            </Row>
          ))}
        </PreferencesGroup>
      )}
    </div>
  )
}
