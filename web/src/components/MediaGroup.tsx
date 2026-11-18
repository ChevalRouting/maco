import { useRef, useState } from 'react'
import { AlertDialog, Button, EmptyState, Meter, PreferencesGroup, Row, fmtBytes } from 'maco-ui'
import { Disc3, HardDrive, Trash2, Upload } from 'lucide-react'
import { deleteMedia, errorMessage, uploadImage, uploadISO, type Media } from '../api'
import { useSession } from '../hooks/useSession'
import { ActionButton } from './ActionButton'
import { JobNotice } from './JobNotice'

type MediaKind = 'image' | 'iso'

interface KindText {
  title: string
  empty: string
  upload: string
  accept: string
}

const text: Record<MediaKind, KindText> = {
  image: { title: 'Custom Images', empty: 'No Custom Images', upload: 'Upload Image…', accept: '.qcow2,.img' },
  iso: { title: 'ISOs', empty: 'No ISOs', upload: 'Upload ISO…', accept: '.iso' },
}

interface MediaGroupProps {
  kind: MediaKind
  items: Media[]
  loaded: boolean
  onChange: () => void
}

export function MediaGroup({ kind, items, loaded, onChange }: MediaGroupProps) {
  const { admin } = useSession()
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [progress, setProgress] = useState(0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState<Media | null>(null)
  const Icon = kind === 'iso' ? Disc3 : HardDrive

  async function upload(selected: File) {
    setFile(selected)
    setBusy(true)
    setError('')
    setProgress(0)
    try {
      await (kind === 'iso' ? uploadISO : uploadImage)(selected, setProgress)
      setFile(null)
      onChange()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    if (!deleting) return
    setBusy(true)
    setError('')
    try {
      await deleteMedia(deleting.id)
      setDeleting(null)
      onChange()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  function chooseFile(event: React.ChangeEvent<HTMLInputElement>) {
    const selected = event.target.files?.[0]
    event.target.value = ''
    if (selected) void upload(selected)
  }

  const uploadButton = admin && (
    <>
      <input ref={input} type="file" accept={text[kind].accept} className="sr-only" tabIndex={-1} aria-hidden="true" onChange={chooseFile} />
      <Button variant="outline" className="gap-2" disabled={busy} onClick={() => input.current?.click()}>
        <Upload aria-hidden="true" className="h-4 w-4" />
        {text[kind].upload}
      </Button>
    </>
  )

  const uploading = file && (
    <Row title={file.name} subtitle={busy ? `${fmtBytes(file.size * progress)} of ${fmtBytes(file.size)}` : 'Upload failed'}>
      {busy ? (
        <Meter value={progress * 100} label={`Upload ${file.name}`} className="w-32" />
      ) : (
        <Button onClick={() => void upload(file)}>Retry</Button>
      )}
    </Row>
  )

  return (
    <div className="space-y-3">
      <JobNotice error={error} />
      {loaded && items.length === 0 && !file ? (
        <EmptyState icon={<Icon />} title={text[kind].empty} action={uploadButton || undefined} />
      ) : (
        <PreferencesGroup title={text[kind].title} header={uploadButton}>
          {uploading}
          {items.map((item) => (
            <Row
              key={item.id}
              prefix={<Icon className="h-5 w-5" />}
              title={item.name}
              subtitle={[kind === 'image' ? `${item.size_gib} GiB` : '', item.in_use ? 'In use' : ''].filter(Boolean).join(' · ') || undefined}
            >
              {admin && (
                <ActionButton
                  label={item.in_use ? `${item.name} Is in Use` : `Delete ${item.name}`}
                  icon={Trash2}
                  disabled={item.in_use || busy}
                  onClick={() => setDeleting(item)}
                />
              )}
            </Row>
          ))}
        </PreferencesGroup>
      )}
      <AlertDialog
        open={!!deleting}
        onCancel={() => setDeleting(null)}
        onConfirm={remove}
        busy={busy}
        destructive
        title={`Delete ${deleting?.name ?? ''}?`}
        description="The file is permanently deleted"
        confirmLabel={kind === 'iso' ? 'Delete ISO' : 'Delete Image'}
      />
    </div>
  )
}
