import { useState } from 'react'
import { AlertDialog, Button, Dialog, PreferencesGroup } from 'maco-ui'
import { listCatalog, listMedia, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { ChoiceRow } from './ChoiceRow'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'

interface ReplaceBootDiskDialogProps {
  vm: VMView
  busy: boolean
  error: string
  onClose: () => void
  onReplace: (imageId: string) => Promise<void>
}

interface ReplacementImage {
  id: string
  name: string
  size_gib: number
  catalog: boolean
  downloaded: boolean
}

export function ReplaceBootDiskDialog({ vm, busy, error, onClose, onReplace }: ReplaceBootDiskDialogProps) {
  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')
  const [imageId, setImageId] = useState('')
  const [confirming, setConfirming] = useState(false)
  const images: ReplacementImage[] = [
    ...catalog.data.map((item) => ({ id: item.id, name: `${item.display_name} · ${item.arch}`, size_gib: 0, catalog: true, downloaded: item.downloaded })),
    ...media.data
      .filter((item) => item.kind === 'image')
      .map((item) => ({ id: `media:${item.id}`, name: item.name, size_gib: item.size_gib, catalog: false, downloaded: true })),
  ]
  const selected = images.find((item) => item.id === imageId)
  const capacity = Math.max(vm.manifest.disk_size_gib, selected?.size_gib || 0)

  async function replace() {
    if (!selected) return
    await onReplace(selected.id)
    setConfirming(false)
  }

  return (
    <>
      <Dialog
        open={!confirming}
        onClose={onClose}
        title="Replace Boot Disk"
        footer={
          <>
            <Button variant="outline" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={busy || !selected} onClick={() => setConfirming(true)}>
              Replace…
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <ResourceNotice resource={catalog} name="images" />
          <ResourceNotice resource={media} name="media" />
          <JobNotice error={error} />
          <PreferencesGroup>
            <ChoiceRow
              title="Image"
              value={imageId}
              choices={images.map((image) => ({ value: image.id, label: image.name }))}
              disabled={busy}
              help={selected ? `${capacity} GiB or more${selected.catalog && !selected.downloaded ? ', downloads first' : ''}` : undefined}
              onChange={setImageId}
            />
          </PreferencesGroup>
        </div>
      </Dialog>
      <AlertDialog
        open={confirming}
        onCancel={() => !busy && setConfirming(false)}
        onConfirm={replace}
        busy={busy}
        destructive
        title="Replace Boot Disk?"
        description={`All data on the boot disk is erased and replaced with ${selected?.name ?? 'the image'}; guest setup runs at the next start`}
        confirmLabel="Replace Disk"
      />
    </>
  )
}
