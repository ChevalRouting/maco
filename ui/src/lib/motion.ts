import { createElement, useLayoutEffect, useRef, useSyncExternalStore } from 'react'

export const EASE_OUT = 'cubic-bezier(0.2, 0.9, 0.25, 1)'
export const EASE_SHEET = 'cubic-bezier(0.32, 0.72, 0, 1)'
export const PHONE_QUERY = '(max-width: 639px)'
export const COMPACT_QUERY = '(max-width: 767px)'

export function reducedMotion(): boolean {
  return (
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

export function matches(query: string): boolean {
  return typeof window !== 'undefined' && window.matchMedia(query).matches
}

export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const media = window.matchMedia(query)
      media.addEventListener('change', onChange)
      return () => media.removeEventListener('change', onChange)
    },
    () => window.matchMedia(query).matches,
    () => false,
  )
}

export type ExitKeyframes = (node: HTMLElement) => Keyframe[]

interface ExitOptions {
  duration: number
  easing?: string
}

function scrollOffsets(node: HTMLElement): Array<[number, number]> {
  const offsets: Array<[number, number]> = []
  node.querySelectorAll('*').forEach((element, index) => {
    if (element.scrollTop > 0) {
      offsets.push([index, element.scrollTop])
    }
  })
  return offsets
}

export function playExit(
  node: HTMLElement | null,
  keyframes: ExitKeyframes,
  { duration, easing = EASE_OUT }: ExitOptions,
) {
  if (!node || !node.isConnected || typeof node.animate !== 'function') {
    return () => {}
  }
  const frames = keyframes(node)
  const offsets = scrollOffsets(node)
  const ghost = node.cloneNode(true) as HTMLElement
  ghost.removeAttribute('id')
  ghost.setAttribute('aria-hidden', 'true')
  ghost.setAttribute('inert', '')
  ghost.style.pointerEvents = 'none'
  ghost.style.animation = 'none'
  return () => {
    document.body.appendChild(ghost)
    const elements = ghost.querySelectorAll('*')
    for (const [index, top] of offsets) {
      const element = elements[index]
      if (element) {
        element.scrollTop = top
      }
    }
    const animation = ghost.animate(frames, {
      duration: reducedMotion() ? Math.min(duration, 120) : duration,
      easing,
      fill: 'forwards',
    })
    const remove = () => ghost.remove()
    animation.finished.then(remove, remove)
  }
}

interface ExitMarkerProps {
  keyframes: ExitKeyframes
  duration: number
  easing?: string
}

export function ExitMarker({ keyframes, duration, easing }: ExitMarkerProps) {
  const marker = useRef<HTMLSpanElement>(null)
  const latest = useRef({ keyframes, duration, easing })
  latest.current = { keyframes, duration, easing }
  const mounted = useRef(false)
  useLayoutEffect(() => {
    mounted.current = true
    const node = marker.current?.parentElement ?? null
    return () => {
      mounted.current = false
      const { keyframes: frames, duration: time, easing: curve } = latest.current
      const start = playExit(node, frames, { duration: time, easing: curve })
      queueMicrotask(() => {
        if (!mounted.current) {
          start()
        }
      })
    }
  }, [])
  return createElement('span', { ref: marker, hidden: true, 'data-exit-marker': '' })
}

export function fadeOut(): Keyframe[] {
  return [{ opacity: 1 }, { opacity: 0 }]
}

export function currentTransform(node: HTMLElement): string {
  const value = getComputedStyle(node).transform
  return value && value !== 'none' ? value : 'translate(0, 0)'
}
