import { PageHeader, SectionNav } from 'maco-ui'
import { listMedia } from '../api'
import { useHasParam } from '../hooks/useHasParam'
import { useResource } from '../hooks/useResource'
import { useTabParam } from '../hooks/useTabParam'
import { CatalogImages } from '../components/CatalogImages'
import { MediaGroup } from '../components/MediaGroup'
import { ResourceNotice } from '../components/ResourceNotice'
import { TemplatesLibrary } from '../components/TemplatesLibrary'

type MediaSection = 'image' | 'iso' | 'template'

interface MediaSectionItem {
  key: MediaSection
  label: string
  group: string
}

const sections: MediaSectionItem[] = [
  { key: 'image', label: 'Images', group: 'Media' },
  { key: 'iso', label: 'ISOs', group: 'Media' },
  { key: 'template', label: 'Templates', group: 'Provisioning' },
]

export default function Media() {
  const media = useResource(listMedia, [], 'media')
  const [section, select] = useTabParam<MediaSection>(sections.map((item) => item.key), 'image')
  const linked = useHasParam('tab')
  const loaded = !media.loading && !media.error

  return (
    <div className="space-y-6">
      <PageHeader title="Media" />
      <ResourceNotice resource={media} name="media" />
      <SectionNav items={sections} active={section} onChange={select} initiallyEntered={linked}>
        {section === 'image' && (
          <div className="mx-auto max-w-3xl space-y-6">
            <CatalogImages />
            <MediaGroup kind="image" items={media.data.filter((m) => m.kind === 'image')} loaded={loaded} onChange={media.refresh} />
          </div>
        )}
        {section === 'iso' && (
          <div className="mx-auto max-w-3xl">
            <MediaGroup kind="iso" items={media.data.filter((m) => m.kind === 'iso')} loaded={loaded} onChange={media.refresh} />
          </div>
        )}
        {section === 'template' && <TemplatesLibrary />}
      </SectionNav>
    </div>
  )
}
