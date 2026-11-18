import { listCatalog, listMedia } from '../api'
import { useResource } from '../hooks/useResource'

interface SourceNameProps {
  image?: string
}

export function SourceName({ image }: SourceNameProps) {
  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')

  if (!image) return <>ISO</>

  const preset = catalog.data.find((item) => item.id === image)
  const custom = media.data.find((item) => `media:${item.id}` === image || item.id === image)

  if (preset) return <>{`${preset.display_name} · ${preset.arch}`}</>
  if (custom) return <>{custom.name}</>
  if (catalog.loading || media.loading) return <>Loading…</>
  return <>{`Unavailable (${image})`}</>
}
