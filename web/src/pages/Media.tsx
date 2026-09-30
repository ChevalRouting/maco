import { PageHeader, PreferencesGroup, Tabs } from 'cheval-ui'
import { listMedia } from '../api'
import { useResource } from '../hooks/useResource'
import { useTabParam } from '../hooks/useTabParam'
import { MediaLibrary } from '../components/MediaLibrary'
import { MediaRow } from '../components/MediaRow'
import { CatalogImages } from '../components/CatalogImages'
import { ResourceNotice } from '../components/ResourceNotice'
import { TemplatesLibrary } from '../components/TemplatesLibrary'

type MediaTab = 'image' | 'iso' | 'template'

const tabs: { key: MediaTab; label: string }[] = [
  { key: 'image', label: 'Images' },
  { key: 'iso', label: 'ISOs' },
  { key: 'template', label: 'Templates' },
]

export default function Media() {
  const media = useResource(listMedia, [], 'media')
  const [tab, setTab] = useTabParam<MediaTab>(['image', 'iso', 'template'], 'image')
  const isos = media.data.filter((m) => m.kind === 'iso')
  const images = media.data.filter((m) => m.kind === 'image')

  return (
    <div className="space-y-6">
      <PageHeader title="Images & ISOs" description="Reusable VM images and installation media." />
      <Tabs id="media" label="Media Type" tabs={tabs} active={tab} onChange={setTab} />
      <ResourceNotice resource={media} name="media library" />

      <div
        role="tabpanel"
        id="media-panel-image"
        aria-labelledby="media-tab-image"
        hidden={tab !== 'image'}
        className="space-y-6"
      >
        <CatalogImages />
        <MediaLibrary mode="image" onChange={media.refresh} />
        {images.length > 0 && (
          <PreferencesGroup title="Custom Images">
            {images.map((m) => (
              <MediaRow key={m.id} media={m} onChange={media.refresh} />
            ))}
          </PreferencesGroup>
        )}
      </div>

      <div
        role="tabpanel"
        id="media-panel-iso"
        aria-labelledby="media-tab-iso"
        hidden={tab !== 'iso'}
        className="space-y-6"
      >
        <MediaLibrary mode="iso" onChange={media.refresh} />
        <PreferencesGroup title="Installation ISOs">
          {isos.map((m) => (
            <MediaRow key={m.id} media={m} onChange={media.refresh} />
          ))}
          {!media.loading && !isos.length && !media.error && (
            <p className="p-4 text-sm text-muted-foreground">
              No ISOs yet. Choose and upload one above.
            </p>
          )}
        </PreferencesGroup>
      </div>

      <div
        role="tabpanel"
        id="media-panel-template"
        aria-labelledby="media-tab-template"
        hidden={tab !== 'template'}
        className="space-y-6"
      >
        <TemplatesLibrary />
      </div>
    </div>
  )
}
