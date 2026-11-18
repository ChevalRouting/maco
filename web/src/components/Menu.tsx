import { useState, type SyntheticEvent } from 'react'
import { MenuButton, type MenuButtonProps } from 'maco-ui'

type MenuAlign = NonNullable<MenuButtonProps['align']>

function roomierSide(event: SyntheticEvent): MenuAlign | null {
  const trigger = (event.target as Element).closest('[aria-haspopup="menu"]')
  if (!trigger) return null
  const rect = trigger.getBoundingClientRect()
  return window.innerWidth - rect.left >= rect.right ? 'start' : 'end'
}

export function Menu(props: Omit<MenuButtonProps, 'align'>) {
  const [align, setAlign] = useState<MenuAlign>('end')

  function measure(event: SyntheticEvent) {
    const side = roomierSide(event)
    if (side) setAlign(side)
  }

  return (
    <span className="contents" onPointerDownCapture={measure} onKeyDownCapture={measure}>
      <MenuButton {...props} align={align} />
    </span>
  )
}
