import { useState } from 'react'
import { Badge, Button, AlertDialog } from 'cheval-ui'
import { Download, Trash2 } from 'lucide-react'
import { listCatalog, downloadCatalogImage, deleteCatalogImage, errorMessage, type CatalogImage } from '../api'
import { useResource } from '../hooks/useResource'
import { DistroIcon } from './DistroIcon'
import { JobNotice } from './JobNotice'
import { ResourceNotice } from './ResourceNotice'
import { useJobAction } from '../hooks/useJobAction'
import { useRowAction } from '../hooks/useRowAction'
import { useSession } from '../hooks/useSession'

function formatSize(bytes: number) {
  if (bytes >= 2 ** 30) {
    return `${(bytes / 2 ** 30).toFixed(1)} GiB`
  }
  return `${Math.max(1, Math.round(bytes / 2 ** 20))} MiB`
}

function CatalogCard({ image, refresh }: { image: CatalogImage; refresh: () => void }) {
  const { admin } = useSession()
  const jobs = useJobAction()
  const action = useRowAction(image.id, image.id, 'image', jobs.run)
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function remove() {
    setBusy(true)
    setError('')
    try {
      await deleteCatalogImage(image.id)
      setConfirm(false)
      refresh()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <article
      className="flex flex-col gap-3 rounded-xl border bg-card p-4"
      aria-label={image.display_name}
    >
      <div className="flex items-start gap-3">
        <DistroIcon distro={image.distro} className="text-4xl text-muted-foreground" />
        <div className="min-w-0 flex-1">
          <h3 className="break-words font-semibold">{image.display_name}</h3>
          <p className="mt-1 break-words text-sm text-muted-foreground">
            {image.description}
          </p>
        </div>
        <Badge variant={image.downloaded ? 'neutral' : 'outline'}>
          {image.downloaded ? 'Ready to Use' : 'Download Required'}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-2" aria-label="Image details">
        <Badge variant="outline">{image.arch}</Badge>
        {image.tags?.map((tag) => (
          <Badge key={tag} variant="outline">{tag}</Badge>
        ))}
      </div>
      {image.downloaded && (
        <p className="text-sm text-muted-foreground">
          Cached file: {formatSize(image.size_bytes)}
        </p>
      )}

      <JobNotice error={error || jobs.error} />

      {admin && (
        <div className="mt-auto flex flex-wrap items-center gap-2">
          {image.downloaded ? (
            <Button
              size="sm"
              variant="outline"
              disabled={busy || action.busy}
              onClick={() => setConfirm(true)}
            >
              <Trash2 aria-hidden="true" className="mr-2 h-4 w-4" />
              Remove Cached Image…
            </Button>
          ) : (
            <Button
              size="sm"
              variant="suggested"
              disabled={action.busy}
              onClick={() => action.execute('image.download', () => downloadCatalogImage(image.id))}
            >
              <Download aria-hidden="true" className="mr-2 h-4 w-4" />
              {action.busy ? 'Downloading…' : 'Download'}
            </Button>
          )}
        </div>
      )}

      <AlertDialog
        open={confirm}
        onCancel={() => setConfirm(false)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Remove ${image.display_name}?`}
        description="The cached source image is permanently deleted. Existing VMs retain their independent disk copies. You can download the source again."
        confirmLabel="Remove Cached Image"
      />
    </article>
  )
}

export function CatalogImages() {
  const catalog = useResource(listCatalog, [], 'catalog')

  return (
    <section className="space-y-3" aria-label="Distributions">
      <div>
        <h2 className="font-semibold">Distributions</h2>
        <p className="text-sm text-muted-foreground">
          Download a cloud image before creating a VM. Each VM receives an independent copy.
        </p>
      </div>

      <ResourceNotice resource={catalog} name="distribution catalog" />

      {catalog.data.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {catalog.data.map((image) => (
            <CatalogCard key={image.id} image={image} refresh={catalog.refresh} />
          ))}
        </div>
      )}

      {!catalog.loading && !catalog.error && !catalog.data.length && (
        <p className="text-sm text-muted-foreground">
          No distributions available. You can upload a custom disk image below.
        </p>
      )}
    </section>
  )
}
