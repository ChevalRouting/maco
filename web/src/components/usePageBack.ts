import { useLocation } from 'react-router-dom'
import { type PageBack } from 'maco-ui'
import { pageLabel, type NavOrigin } from '../ux'

interface OriginState {
  from?: NavOrigin
}

export function usePageBack(to: string, label: string): PageBack {
  const from = (useLocation().state as OriginState | null)?.from
  return from ? { to: from.pathname + from.search, label: pageLabel(from.pathname) } : { to, label }
}
