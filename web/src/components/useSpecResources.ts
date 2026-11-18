import { getHost, listCatalog, listMedia, listNetworks } from '../api'
import { useResource } from '../hooks/useResource'
import { type SpecResources } from './vmSpec'

export function useSpecResources() {
  const catalog = useResource(listCatalog, [], 'catalog')
  const media = useResource(listMedia, [], 'media')
  const networks = useResource(listNetworks, [], 'networks')
  const host = useResource(getHost, null, 'host')

  const resources: SpecResources = {
    catalog: catalog.data,
    media: media.data,
    networks: networks.data,
    networksLoading: networks.loading,
  }
  const loading = catalog.loading || media.loading || networks.loading
  const notices = [
    { resource: catalog, name: 'images' },
    { resource: media, name: 'media' },
    { resource: networks, name: 'networks' },
  ]

  return { resources, host: host.data, loading, notices }
}
