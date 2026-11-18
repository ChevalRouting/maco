import { useSyncExternalStore } from 'react'

function useMedia(query: string): boolean {
  function subscribe(onChange: () => void) {
    const media = window.matchMedia(query)
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }

  return useSyncExternalStore(subscribe, () => window.matchMedia(query).matches)
}

export function useNarrow(): boolean {
  return useMedia('(max-width: 639px)')
}

export function useCompact(): boolean {
  return useMedia('(max-width: 767px)')
}
