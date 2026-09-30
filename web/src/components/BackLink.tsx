import { useLocation } from 'react-router-dom'
import { BackLink as SharedBackLink } from 'cheval-ui'
import { pageLabel, type NavOrigin } from '../ux'

interface BackLinkProps {
  to: string
  children: React.ReactNode
}

export function BackLink({ to, children }: BackLinkProps) {
  const from = (useLocation().state as { from?: NavOrigin } | null)?.from
  const target = from ? from.pathname + from.search : to
  const label = from ? `Back to ${pageLabel(from.pathname)}` : children
  return <SharedBackLink to={target}>{label}</SharedBackLink>
}
