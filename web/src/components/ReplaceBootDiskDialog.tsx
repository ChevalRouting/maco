import { useState } from 'react'
import {
  AlertDialog,
  Button,
  Dialog,
  NoticeBanner,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from 'cheval-ui'
import { listCatalog, listMedia, type VMView } from '../api'
import { useResource } from '../hooks/useResource'
import { ResourceNotice } from './ResourceNotice'
import { JobNotice } from './JobNotice'

interface ReplaceBootDiskDialogProps {
  vm: VMView
  busy: boolean
  error: string
  onClose: () => void
  onReplace: (imageId: string) => Promise<void>
}

export function ReplaceBootDiskDialog({
  vm,
  busy,
  error,
  onClose,
  onReplace,
}: ReplaceBootDiskDialogProps) {
  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')
  const [imageId, setImageId] = useState('')
  const [confirming, setConfirming] = useState(false)
  const images = [
    ...catalog.data.map((item) => ({
      id: item.id,
      name: `${item.display_name} · ${item.arch}`,
      size_gib: 0,
      catalog: true,
      downloaded: item.downloaded,
    })),
    ...media.data
      .filter((item) => item.kind === 'image')
      .map((item) => ({
        ...item,
        id: `media:${item.id}`,
        catalog: false,
        downloaded: true,
      })),
  ]
  const selected = images.find((item) => item.id === imageId)
  const blocked = busy || vm.phase === 'running'

  return (
    <>
      <Dialog
        open={!confirming}
        onClose={onClose}
        title="Replace Boot Disk from Image"
        className="max-w-lg [&>div:nth-child(2)]:pt-0"
      >
        <div className="space-y-4">
          <NoticeBanner intent="warning">
            This permanently erases the current boot disk. Other disks stay attached.
            The saved guest setup will be supplied on the next start if enabled; the image must support its provisioner.
          </NoticeBanner>
          <ResourceNotice resource={catalog} name="Linux image catalog" />
          <ResourceNotice resource={media} name="disk images" />
          <JobNotice error={error} />
          {!media.loading && !catalog.loading && !media.error && !catalog.error && images.length === 0 && (
            <p className="text-sm text-muted-foreground">
              No disk images are available. Upload or create one in Media.
            </p>
          )}
          <Select value={imageId} onValueChange={setImageId} disabled={blocked}>
            <SelectTrigger aria-label="Replacement disk image">
              <SelectValue placeholder="Select a disk image" />
            </SelectTrigger>
            <SelectContent className="z-[60]">
              {images.map((image) => (
                <SelectItem key={image.id} value={image.id}>
                  {image.name}
                  {image.catalog ? ' · Linux catalog' : ` · ${image.size_gib} GiB`}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {selected?.catalog && !selected.downloaded && (
            <p className="text-sm text-muted-foreground">
              This image will download during replacement. Follow progress in Activity.
            </p>
          )}
          {selected && (
            <p className="text-sm text-muted-foreground">
              {selected.catalog
                ? `Capacity stays at least ${vm.manifest.disk_size_gib} GiB and grows if the image needs more space.`
                : `New capacity: ${Math.max(vm.manifest.disk_size_gib, selected.size_gib)} GiB.`}{' '}
              The fresh disk will boot first.
            </p>
          )}
          {vm.phase === 'running' && (
            <NoticeBanner intent="info">
              Stop the VM before replacing its boot disk.
            </NoticeBanner>
          )}
          <div className="flex justify-end gap-2">
            <Button disabled={busy} onClick={onClose}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={blocked || !selected}
              onClick={() => setConfirming(true)}
            >
              Replace Disk…
            </Button>
          </div>
        </div>
      </Dialog>
      <AlertDialog
        open={confirming}
        onCancel={() => {
          if (!busy) setConfirming(false)
        }}
        onConfirm={async () => {
          if (blocked || !selected) return
          await onReplace(selected.id)
          setConfirming(false)
        }}
        busy={busy}
        destructive
        title={`Replace ${vm.manifest.name}'s boot disk?`}
        description={`All data on the current boot disk will be permanently erased and replaced with a fresh copy of ${selected?.name ?? 'the selected image'}. Start the VM afterward to run its enabled guest setup.`}
        confirmLabel="Erase and Replace Disk"
      />
    </>
  )
}
