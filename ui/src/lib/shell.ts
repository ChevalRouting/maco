import { createContext, createElement, useContext, useSyncExternalStore } from 'react'

export interface ShellContextValue {
  toolbar: HTMLElement | null
  root: HTMLElement | null
  scroller: HTMLElement | null
  navActions: HTMLElement | null
  setNavActions: (node: HTMLElement | null) => void
}

export const ShellContext = createContext<ShellContextValue>({
  toolbar: null,
  root: null,
  scroller: null,
  navActions: null,
  setNavActions: () => {},
})

export function useShell() {
  return useContext(ShellContext)
}

export interface PageHeaderInfo {
  title: string
  backLabel: string | null
}

let current: PageHeaderInfo | null = null
const listeners = new Set<() => void>()

export function setPageHeader(info: PageHeaderInfo | null) {
  if (current?.title === info?.title && current?.backLabel === info?.backLabel) {
    return
  }
  current = info
  listeners.forEach((listener) => listener())
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function usePageHeader() {
  return useSyncExternalStore(
    subscribe,
    () => current,
    () => null,
  )
}

export function setShellFlag(
  root: HTMLElement | null,
  flag: 'scrolled' | 'collapsed',
  on: boolean,
) {
  if (!root) {
    return
  }
  const name = `data-${flag}`
  if (on) {
    root.setAttribute(name, '')
  } else {
    root.removeAttribute(name)
  }
}

export function NavBarActions({ className }: { className?: string }) {
  const { setNavActions } = useShell()
  return createElement('div', { ref: setNavActions, className })
}
