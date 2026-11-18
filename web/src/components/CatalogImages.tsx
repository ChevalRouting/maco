import { useState } from 'react'
import { AlertDialog, PreferencesGroup, Row } from 'maco-ui'
import { Download, Trash2 } from 'lucide-react'
import { listCatalog, downloadCatalogImage, deleteCatalogImage, errorMessage, type CatalogImage } from '../api'
import { useResource } from '../hooks/useResource'
import { useJobAction } from '../hooks/useJobAction'
import { useRowAction } from '../hooks/useRowAction'
import { useSession } from '../hooks/useSession'
import { formatBytes } from '../ux'
import { ActionButton } from './ActionButton'
import { DistroIcon } from './DistroIcon'
import { ResourceNotice } from './ResourceNotice'

interface CatalogRowProps {
  image: CatalogImage
  refresh: () => void
}

function CatalogRow({ image, refresh }: CatalogRowProps) {
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

  const detail = [image.arch, image.downloaded ? formatBytes(image.size_bytes) : 'Not downloaded'].join(' · ')

  return (
    <>
      <Row
        prefix={<DistroIcon distro={image.distro} className="inline-block w-6 text-center text-2xl" />}
        title={image.display_name}
        subtitle={error || jobs.error ? <span className="text-danger">{error || jobs.error}</span> : detail}
      >
        {admin &&
          (image.downloaded ? (
            <ActionButton label={`Delete ${image.display_name}`} icon={Trash2} busy={busy} onClick={() => setConfirm(true)} />
          ) : (
            <ActionButton
              label={`Download ${image.display_name}`}
              icon={Download}
              busy={action.busy}
              onClick={() => action.execute('image.download', () => downloadCatalogImage(image.id))}
            />
          ))}
      </Row>
      <AlertDialog
        open={confirm}
        onCancel={() => setConfirm(false)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Delete ${image.display_name}?`}
        description="Existing virtual machines keep their own disk copies"
        confirmLabel="Delete Image"
      />
    </>
  )
}

export function CatalogImages() {
  const catalog = useResource(listCatalog, [], 'catalog')

  return (
    <div className="space-y-3">
      <ResourceNotice resource={catalog} name="distributions" />
      {catalog.data.length > 0 && (
        <PreferencesGroup title="Distributions">
          {catalog.data.map((image) => (
            <CatalogRow key={image.id} image={image} refresh={catalog.refresh} />
          ))}
        </PreferencesGroup>
      )}
    </div>
  )
}
