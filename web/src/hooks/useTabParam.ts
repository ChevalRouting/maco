import { useSearchParams } from 'react-router-dom'

export function useTabParam<T extends string>(keys: readonly T[], fallback: T, param = 'tab') {
  const [searchParams, setSearchParams] = useSearchParams()
  const active = keys.find((key) => key === searchParams.get(param)) || fallback

  function select(value: T) {
    if (value === active) return

    const next = new URLSearchParams(searchParams)
    next.set(param, value)
    setSearchParams(next, { replace: true })
  }

  return [active, select] as const
}
