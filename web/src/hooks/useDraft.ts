import { useEffect, useState } from 'react'

export function useDraft<T>(key: string, value: T) {
  const persisted = JSON.stringify(value)
  const [state, setState] = useState<{ base: string; value: T }>(() => {
    try { const saved = JSON.parse(sessionStorage.getItem(key) || 'null'); return saved && typeof saved.base === 'string' && saved.value && Object.keys(value as object).every(field => field in saved.value) ? saved : { base: persisted, value } } catch { return { base: persisted, value } }
  })
  const dirty = JSON.stringify(state.value) !== persisted
  const conflict = dirty && state.base !== persisted
  useEffect(() => {
    if ((JSON.stringify(state.value) === state.base || JSON.stringify(state.value) === persisted) && state.base !== persisted) setState({ base: persisted, value: JSON.parse(persisted) })
  }, [persisted, state])
  useEffect(() => {
    try { if (dirty) sessionStorage.setItem(key, JSON.stringify(state)); else sessionStorage.removeItem(key) } catch { void 0 }
  }, [key, state, dirty])
  return { value: state.value, dirty, conflict, set: (next: T) => setState(current => ({ ...current, value: next })), discard: () => setState({ base: persisted, value: JSON.parse(persisted) }) }
}
