const variables: { token: string; description: string }[] = [
  { token: '[[ .Name ]]', description: "The VM's name (also its default hostname)." },
  { token: '[[ .Hostname ]]', description: 'Guest hostname; falls back to the VM name.' },
  { token: '[[ .CPUs ]]', description: 'Virtual CPU count.' },
  { token: '[[ .MemoryMiB ]]', description: 'Memory in MiB.' },
  { token: '[[ .DiskSizeGiB ]]', description: 'Boot disk capacity in GiB.' },
  {
    token: '[[ range .SSHKeys ]] [[ . ]] [[ end ]]',
    description: "SSH public keys from the creating account's profile.",
  },
  { token: '[[ .Network ]]', description: 'Default network ID.' },
  { token: '[[ .Addresses ]] · [[ .Gateway ]] · [[ .Nameservers ]]', description: 'Primary interface networking.' },
  { token: '[[ range .Disks ]] [[ .Name ]] [[ .SizeGiB ]] [[ end ]]', description: 'Iterate additional data disks.' },
  { token: '[[ range .Interfaces ]] [[ .Network ]] [[ .MAC ]] [[ end ]]', description: 'Iterate network interfaces.' },
]

export function TemplateVariables() {
  return (
    <details className="rounded-xl border p-4">
      <summary className="cursor-pointer text-sm font-semibold">Available guest variables</summary>
      <p className="mt-2 text-xs text-muted-foreground">
        Reference these in the guest configuration below. They are evaluated when the VM is created.
        Content without <code>[[ ]]</code> is delivered unchanged, so cloud-init Jinja{' '}
        <code>{'{{ }}'}</code> is safe.
      </p>
      <table className="mt-3 w-full border-collapse text-xs">
        <thead>
          <tr className="border-b text-left text-muted-foreground">
            <th className="py-1 pr-4 font-medium">Variable</th>
            <th className="py-1 font-medium">Description</th>
          </tr>
        </thead>
        <tbody>
          {variables.map((item) => (
            <tr key={item.token} className="border-b last:border-0 align-top">
              <td className="py-1 pr-4">
                <code className="whitespace-pre-wrap break-words">{item.token}</code>
              </td>
              <td className="py-1 text-muted-foreground">{item.description}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  )
}
