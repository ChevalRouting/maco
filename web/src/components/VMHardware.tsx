import type { FormEvent } from 'react'
import { EntryRow, NoticeBanner, PreferencesGroup, TagInput } from 'maco-ui'
import { getHost, updateHardware, type VMView, type Job } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { useResource } from '../hooks/useResource'
import { useDraft } from '../hooks/useDraft'
import { memoryLabel, nameError } from '../ux'
import { FormActions } from './FormActions'
import { IntegerEntryRow } from './IntegerEntryRow'
import { JobNotice } from './JobNotice'
import { MemoryEntryRow } from './MemoryEntryRow'

interface VMHardwareProps {
  vm: VMView
  run: (action: () => Promise<Job>) => Promise<Job | null>
}

export function VMHardware({ vm, run }: VMHardwareProps) {
  const manifest = vm.manifest
  const draft = useDraft(`maco:draft:${manifest.id}:general`, {
    name: manifest.name,
    cpus: manifest.cpus,
    memory_mib: manifest.memory_mib,
    tags: manifest.tags ?? [],
  })
  const host = useResource(getHost, null, 'host')
  const action = useRowAction(manifest.id, manifest.name, 'vm', run)
  const running = vm.phase === 'running'
  const invalidName = nameError(draft.value.name)

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || !draft.dirty || draft.conflict || invalidName) return
    const body = running ? { name: draft.value.name, tags: draft.value.tags } : draft.value
    await action.execute('vm.hardware', () => updateHardware(manifest.id, body))
  }

  const hostMemoryMib = host.data ? host.data.memory_bytes / 1048576 : 0
  const overCapacity = host.data && (draft.value.cpus > host.data.cpus || draft.value.memory_mib > hostMemoryMib)

  return (
    <form onSubmit={save} className="max-w-3xl space-y-6">
      <JobNotice error={action.error} />
      {draft.conflict && <NoticeBanner intent="warning">Changed elsewhere; discard to load the saved values</NoticeBanner>}
      {running && <NoticeBanner intent="info">Shut down to change vCPUs and memory</NoticeBanner>}

      <fieldset disabled={action.busy}>
        <PreferencesGroup title="Identity" description={`ID ${manifest.id}`}>
          <EntryRow
            id="hardware-name"
            title="Name"
            required
            value={draft.value.name}
            error={invalidName}
            onChange={(event) => draft.set({ ...draft.value, name: event.target.value })}
          />
        </PreferencesGroup>
      </fieldset>

      <fieldset disabled={running || action.busy}>
        <PreferencesGroup title="Resources">
          <IntegerEntryRow
            id="hardware-cpus"
            title="vCPUs"
            min={1}
            required
            value={draft.value.cpus}
            help={host.data ? `Host has ${host.data.cpus}` : undefined}
            onValueChange={(cpus) => draft.set({ ...draft.value, cpus })}
          />
          <MemoryEntryRow
            id="hardware-memory"
            value={draft.value.memory_mib}
            help={host.data ? `Host has ${memoryLabel(hostMemoryMib)}` : undefined}
            onValueChange={(memory_mib) => draft.set({ ...draft.value, memory_mib })}
          />
        </PreferencesGroup>
      </fieldset>
      {overCapacity && <NoticeBanner intent="warning">Exceeds the host’s CPUs or memory</NoticeBanner>}

      <PreferencesGroup title="Tags">
        <div className="px-4 py-3">
          <TagInput values={draft.value.tags} placeholder="Add a tag" onChange={(tags) => draft.set({ ...draft.value, tags })} />
        </div>
      </PreferencesGroup>

      <FormActions dirty={draft.dirty} busy={action.busy} disabled={draft.conflict || !!invalidName} onDiscard={draft.discard} />
    </form>
  )
}
