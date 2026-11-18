import { useSyncExternalStore } from 'react'

let current: string | null = null
const listeners = new Set<() => void>()

export function setSectionTitle(title: string | null) {
  if (current === title) {
    return
  }
  current = title
  listeners.forEach((listener) => listener())
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function useSectionTitle() {
  return useSyncExternalStore(subscribe, () => current, () => null)
}
