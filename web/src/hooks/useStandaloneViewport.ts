import { useLayoutEffect } from 'react'

export function useStandaloneViewport() {
  useLayoutEffect(() => {
    const standalone = window.matchMedia('(display-mode: standalone)').matches
      || ('standalone' in navigator && navigator.standalone === true)
    if (!standalone) return

    const root = document.documentElement
    const viewport = document.querySelector<HTMLMetaElement>('meta[name="viewport"]')
    const previousContent = viewport?.content
    root.classList.add('standalone-app')
    if (viewport) {
      viewport.content = 'width=device-width, initial-scale=1, minimum-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover'
    }

    const preventPageZoom = (event: Event) => event.preventDefault()
    document.addEventListener('gesturestart', preventPageZoom, { passive: false })
    document.addEventListener('gesturechange', preventPageZoom, { passive: false })

    return () => {
      document.removeEventListener('gesturestart', preventPageZoom)
      document.removeEventListener('gesturechange', preventPageZoom)
      root.classList.remove('standalone-app')
      if (viewport && previousContent !== undefined) viewport.content = previousContent
    }
  }, [])
}
