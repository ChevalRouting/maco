import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { cn, useTheme, type MenuItem, type ThemePreference } from 'maco-ui'
import { CircleUser, Info, LogOut, Monitor, Moon, SlidersHorizontal, Sun } from 'lucide-react'
import { useSession } from '../../hooks/useSession'
import { AboutDialog } from './AboutDialog'
import { signOut } from './signOut'
import { Menu } from '../Menu'

interface AppearanceOption {
  value: ThemePreference
  label: string
  icon: MenuItem['icon']
}

export const APPEARANCE_OPTIONS: AppearanceOption[] = [
  { value: 'system', label: 'Follow System', icon: Monitor },
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
]

interface MainMenuProps {
  collapsed: boolean
}

export function MainMenu({ collapsed }: MainMenuProps) {
  const navigate = useNavigate()
  const session = useSession()
  const { preference, setPreference } = useTheme()
  const [aboutOpen, setAboutOpen] = useState(false)
  const name = session.user?.username || 'Account'

  const sections: MenuItem[][] = [
    APPEARANCE_OPTIONS.map((option) => ({
      key: `appearance-${option.value}`,
      label: option.label,
      icon: option.icon,
      checked: preference === option.value,
      onSelect: () => setPreference(option.value),
    })),
    [
      { key: 'preferences', label: 'Preferences', icon: SlidersHorizontal, onSelect: () => navigate('/preferences') },
      { key: 'about', label: 'About maco', icon: Info, onSelect: () => setAboutOpen(true) },
    ],
    [{ key: 'sign-out', label: 'Sign Out', icon: LogOut, onSelect: signOut }],
  ]

  return (
    <>
      <Menu
        label="Main Menu"
        sections={sections}
        className={cn(
          'w-full text-sidebar-foreground/80 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground',
          collapsed ? 'h-8' : 'h-8 justify-start px-3',
        )}
      >
        <CircleUser className="h-4 w-4 shrink-0" />
        {!collapsed && <span className="truncate">{name}</span>}
      </Menu>
      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </>
  )
}
