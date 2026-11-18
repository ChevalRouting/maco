import { Row, SwitchRow } from 'maco-ui'
import { type NetworkInterface } from '../api'

interface InterfaceMembersProps {
  interfaces: NetworkInterface[]
  selected: string[]
  onChange: (members: string[]) => void
}

export function InterfaceMembers({ interfaces, selected, onChange }: InterfaceMembersProps) {
  function toggle(device: string, on: boolean) {
    onChange(on ? [...selected, device] : selected.filter((entry) => entry !== device))
  }

  if (interfaces.length === 0) return <Row title="Members" subtitle="No host interfaces" />

  return (
    <>
      {interfaces.map((iface) => (
        <SwitchRow
          key={iface.device}
          title={iface.device}
          subtitle={iface.hardware_port || 'Not detected'}
          checked={selected.includes(iface.device)}
          onCheckedChange={(on) => toggle(iface.device, on)}
        />
      ))}
    </>
  )
}
