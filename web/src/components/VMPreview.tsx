import { useEffect, useState } from 'react'
import { Monitor } from 'lucide-react'
import { getVMPreview } from '../api'
import { subscribeResourceEvents } from '../hooks/resourceEvents'

interface VMPreviewProps {
  id: string
  name: string
  running: boolean
  bootTime: number
  large?: boolean
}

export function VMPreview({ id, name, running, bootTime, large }: VMPreviewProps) {
  const [image, setImage] = useState('')
  const [imageBoot, setImageBoot] = useState(0)
  const [ratio, setRatio] = useState(8 / 5)

  function subscribe() {
    setImage('')
    setRatio(8 / 5)
    if (!running || bootTime <= 0) return

    const controller = new AbortController()
    let current = ''
    let loading = false
    let dirty = false

    async function refresh() {
      if (loading) {
        dirty = true
        return
      }
      dirty = false

      loading = true
      try {
        const { data: blob, response } = await getVMPreview(id, controller.signal)
        if (!response?.ok || !blob) return

        const captureHeader = response.headers.get('X-Preview-Captured-At')
        const capturedAt = captureHeader ? Number(captureHeader) : Date.parse(response.headers.get('Last-Modified') || '')
        if (!Number.isFinite(capturedAt) || capturedAt < bootTime * 1000) return

        if (controller.signal.aborted) return

        const next = URL.createObjectURL(blob)
        if (current) URL.revokeObjectURL(current)
        current = next
        setImage(next)
        setImageBoot(bootTime)
      } catch {
        return
      } finally {
        loading = false
        if (!controller.signal.aborted && dirty) void refresh()
      }
    }

    const unsubscribe = subscribeResourceEvents((resources) => {
      if (resources.includes('all') || resources.includes(`preview:${id}`)) void refresh()
    })
    void refresh()
    return () => {
      controller.abort()
      unsubscribe()
      if (current) URL.revokeObjectURL(current)
    }
  }

  useEffect(subscribe, [id, running, bootTime])

  function measure(event: React.SyntheticEvent<HTMLImageElement>) {
    const element = event.currentTarget
    if (element.naturalHeight) setRatio(element.naturalWidth / element.naturalHeight)
  }

  return (
    <div
      className={`relative flex items-center justify-center overflow-hidden rounded-lg bg-muted ${large ? 'w-full' : 'w-24'}`}
      style={{ aspectRatio: running ? ratio : 8 / 5 }}
    >
      {running && image && imageBoot === bootTime ? (
        <img src={image} alt={`${name} display`} className="absolute inset-0 h-full w-full object-contain" onLoad={measure} />
      ) : (
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <Monitor aria-hidden="true" className={large ? 'h-8 w-8' : 'h-5 w-5'} />
          {large && <p className="text-sm">{running ? 'No display yet' : 'Stopped'}</p>}
        </div>
      )}
    </div>
  )
}
