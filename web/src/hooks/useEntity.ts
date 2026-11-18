import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useResource } from './useResource'

export function useEntity<T>(load: () => Promise<T | null>, domain: string, fallback: string) {
  const navigate = useNavigate()
  const resource = useResource<T | null>(load, null, domain)
  useEffect(() => {
    if (!resource.loading && resource.data === null && !resource.error) navigate(fallback, { replace: true })
  }, [resource.loading, resource.data, resource.error, navigate, fallback])
  return resource
}
