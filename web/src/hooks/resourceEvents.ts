type Listener = (resources: string[]) => void
const listeners = new Set<Listener>()
export function subscribeResourceEvents(listener: Listener) {
  listeners.add(listener)
  return () => { listeners.delete(listener) }
}
export function notifyResources(resources: string[]) {
  for (const listener of listeners) listener(resources)
}
