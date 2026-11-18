import { Table, TableBody, TableCell, TableRow } from 'maco-ui'
import { ChevronRight } from 'lucide-react'

interface TemplateVariable {
  token: string
  description: string
}

const variables: TemplateVariable[] = [
  { token: '[[ .Name ]]', description: 'VM name, the default hostname' },
  { token: '[[ .Hostname ]]', description: 'Guest hostname, or the VM name' },
  { token: '[[ .CPUs ]]', description: 'vCPU count' },
  { token: '[[ .MemoryMiB ]]', description: 'Memory in MiB' },
  { token: '[[ .DiskSizeGiB ]]', description: 'Boot disk capacity in GiB' },
  { token: '[[ range .SSHKeys ]] [[ . ]] [[ end ]]', description: 'SSH public keys from Preferences' },
  { token: '[[ .Network ]]', description: 'Default network ID' },
  { token: '[[ .Addresses ]] [[ .Gateway ]] [[ .Nameservers ]]', description: 'Primary adapter addressing' },
  { token: '[[ range .Disks ]] [[ .Name ]] [[ .SizeGiB ]] [[ end ]]', description: 'Additional disks' },
  { token: '[[ range .Interfaces ]] [[ .Network ]] [[ .MAC ]] [[ end ]]', description: 'Network adapters' },
]

export function TemplateVariables() {
  return (
    <details className="group">
      <summary className="flex min-h-[2.75rem] cursor-pointer list-none items-center gap-3 px-4 text-sm active:bg-foreground/[0.05] [&::-webkit-details-marker]:hidden">
        <span className="flex-1">Variables</span>
        <ChevronRight aria-hidden="true" strokeWidth={2.5} className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60 transition-transform group-open:rotate-90" />
      </summary>
      <div className="px-4 pb-3">
        <Table cards={false}>
          <TableBody>
            {variables.map((item) => (
              <TableRow key={item.token}>
                <TableCell className="whitespace-pre-wrap break-words align-top font-mono text-xs">{item.token}</TableCell>
                <TableCell className="align-top text-xs text-muted-foreground">{item.description}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </details>
  )
}
