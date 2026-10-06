import type { FormEvent } from 'react'
import { Button, NoticeBanner, PreferencesGroup, TagInput } from 'cheval-ui'
import { getHost, updateHardware, type VMView, type Job } from '../api'
import { useRowAction } from '../hooks/useRowAction'
import { useResource } from '../hooks/useResource'
import { useDraft } from '../hooks/useDraft'
import { IntegerEntryRow } from './IntegerEntryRow'
import { MemoryEntryRow } from './MemoryEntryRow'
import { ResourceNotice } from './ResourceNotice'
import { memoryLabel } from '../ux'

export function VMHardware({ vm, run }: { vm: VMView; run: (action: () => Promise<Job>) => Promise<Job | null> }) {
  const manifest = vm.manifest
  const draft = useDraft(`maco:draft:${manifest.id}:general`, {
    cpus: manifest.cpus,
    memory_mib: manifest.memory_mib,
    tags: manifest.tags ?? [],
  })
  const host = useResource(getHost, null, 'host')
  const action = useRowAction(manifest.id, manifest.name, 'vm', run)
  const running = vm.phase === 'running'
  const computeBlocked = running || action.busy

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (action.busy || !draft.dirty || draft.conflict) return
    const body = running ? { tags: draft.value.tags } : draft.value
    await action.execute('vm.hardware', () => updateHardware(manifest.id, body))
  }

  const hostMemoryMib = host.data ? host.data.memory_bytes / (1024 * 1024) : 0
  const overCapacity =
    host.data && (draft.value.cpus > host.data.cpus || draft.value.memory_mib > hostMemoryMib)

  return (
    <form onSubmit={save} className="max-w-2xl space-y-4">
      <ResourceNotice resource={host} name="host capacity" />
      {host.data && (
        <p className="text-sm text-muted-foreground">
          Host capacity: {host.data.cpus} CPUs · {memoryLabel(hostMemoryMib)}. These are allocations,
          not measurements of guest usage.
        </p>
      )}
      {overCapacity && (
        <NoticeBanner intent="warning">
          This allocation exceeds the host’s physical capacity and can reduce performance or prevent
          startup.
        </NoticeBanner>
      )}
      {draft.conflict && (
        <NoticeBanner intent="warning">
          Saved settings changed while you were editing. Discard this draft to load the latest values
          before saving.
        </NoticeBanner>
      )}

      <fieldset disabled={computeBlocked}>
        <PreferencesGroup
          title="CPU & Memory"
          description={running ? 'Stop the VM to change CPU or memory.' : undefined}
        >
          <IntegerEntryRow
            id="hardware-cpus"
            title="CPUs"
            min={1}
            required
            value={draft.value.cpus}
            onValueChange={(cpus) => draft.set({ ...draft.value, cpus })}
          />
          <MemoryEntryRow
            id="hardware-memory"
            value={draft.value.memory_mib}
            onValueChange={(memory_mib) => draft.set({ ...draft.value, memory_mib })}
          />
        </PreferencesGroup>
      </fieldset>

      <PreferencesGroup
        title="Tags"
        description="Freeform labels for grouping and filtering, such as role or environment."
      >
        <div className="px-4 py-3">
          <TagInput
            values={draft.value.tags}
            placeholder="Add a tag"
            onChange={(tags) => draft.set({ ...draft.value, tags })}
          />
        </div>
      </PreferencesGroup>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="submit" variant="suggested" disabled={action.busy || !draft.dirty || draft.conflict}>
          {action.busy ? 'Saving…' : 'Save'}
        </Button>
        <Button type="button" variant="outline" disabled={!draft.dirty || action.busy} onClick={draft.discard}>
          Discard Changes
        </Button>
        <span role="status" className="text-sm text-muted-foreground">
          {draft.dirty ? 'Unsaved changes · Draft kept in this browser tab' : 'Matches saved settings'}
        </span>
      </div>

    </form>
  )
}
