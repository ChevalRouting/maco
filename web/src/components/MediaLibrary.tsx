import { useId, useRef, useState } from 'react'
import { Button, NoticeBanner, fmtBytes } from 'cheval-ui'
import { uploadISO, uploadImage, errorMessage } from '../api'
import { useSession } from '../hooks/useSession'
import { JobNotice } from './JobNotice'

const config = {
  iso: {
    title: 'Upload Installation ISO',
    description: 'Choose an ARM64 ISO, up to 10 GiB.',
    accept: '.iso',
    label: 'Choose ISO…',
    upload: uploadISO,
  },
  image: {
    title: 'Upload Custom Disk Image',
    description: 'Choose an ARM64 .qcow2 or .img. New VMs receive an independent copy.',
    accept: '.qcow2,.img',
    label: 'Choose Image…',
    upload: uploadImage,
  },
}

interface MediaLibraryProps {
  mode?: 'image' | 'iso'
  onChange?: () => void
  onBusyChange?: (busy: boolean) => void
}

export function MediaLibrary({ mode = 'image', onChange, onBusyChange }: MediaLibraryProps) {
  const { admin } = useSession()
  const id = useId()
  const input = useRef<HTMLInputElement>(null)
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [progress, setProgress] = useState(0)
  const [saved, setSaved] = useState('')
  const { title, description, accept, label, upload } = config[mode]

  if (!admin) return null

  async function perform(selectedFile: File) {
    if (busy) return
    setFile(selectedFile)
    setBusy(true)
    onBusyChange?.(true)
    setError('')
    setSaved('')
    setProgress(0)
    try {
      await upload(selectedFile, setProgress)
      setSaved(`${selectedFile.name} is available in the library.`)
      setFile(null)
      onChange?.()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
      onBusyChange?.(false)
    }
  }

  const percent = Math.round(progress * 100)

  return (
    <section className="space-y-3" aria-labelledby={`${id}-title`}>
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h2 id={`${id}-title`} className="font-semibold">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <input
          ref={input}
          id={id}
          aria-label={label}
          type="file"
          accept={accept}
          disabled={busy}
          className="sr-only"
          onChange={(event) => {
            const selectedFile = event.target.files?.[0]
            event.target.value = ''
            if (selectedFile) void perform(selectedFile)
          }}
        />

        <Button type="button" className="shrink-0" disabled={busy} onClick={() => input.current?.click()}>
          {label}
        </Button>
      </div>

      {file && (
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3 text-sm">
            <p className="min-w-0 break-words">{file.name} · {fmtBytes(file.size)}</p>
            {busy && <span className="shrink-0 tabular-nums text-muted-foreground">{percent}%</span>}
            {!busy && error && (
              <Button type="button" size="sm" className="shrink-0" onClick={() => void perform(file)}>
                Retry Upload
              </Button>
            )}
          </div>

          {busy && (
            <div className="space-y-2">
              <div
                role="progressbar"
                aria-label={`Upload ${file.name}`}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={percent}
                className="h-2 w-full overflow-hidden rounded-full bg-muted"
              >
                <div className="h-full bg-primary" style={{ width: `${percent}%` }} />
              </div>
              <p role="status" className="text-sm text-muted-foreground">
                {progress < 1
                  ? `${fmtBytes(file.size * progress)} of ${fmtBytes(file.size)} uploaded`
                  : 'Transfer complete. Finalizing the library entry…'}
              </p>
            </div>
          )}
        </div>
      )}
      <JobNotice error={error} />
      {saved && <NoticeBanner intent="success">{saved}</NoticeBanner>}
    </section>
  )
}
