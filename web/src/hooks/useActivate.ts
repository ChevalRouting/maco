import { type MouseEvent } from 'react'
import { useNavigate } from 'react-router-dom'

const interactive = 'a, button, input, select, textarea, label, [role="menu"], [role="dialog"], [role="alertdialog"]'
const activeRow = 'cursor-pointer active:bg-foreground/[0.05]'

export interface Activation {
  onClick?: (event: MouseEvent<HTMLElement>) => void
  className?: string
}

export function activatable(activate?: () => void): Activation {
  if (!activate) return {}

  function onClick(event: MouseEvent<HTMLElement>) {
    const target = event.target as Element
    if (!event.currentTarget.contains(target) || target.closest(interactive)) return
    if (window.getSelection()?.toString()) return
    activate?.()
  }

  return { onClick, className: activeRow }
}

export function useRowLink(to: string, state?: unknown): Activation {
  const navigate = useNavigate()
  return activatable(() => navigate(to, { state }))
}
