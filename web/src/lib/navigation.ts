import { Code, HardDrive, LayoutDashboard, ListTodo, Network, Disc3, Server, ShieldCheck, SquareTerminal } from 'lucide-react'
import type { NavGroup, NavItem } from 'maco-ui'

export function navigation(admin: boolean): NavGroup[] {
  return [
    {
      items: [
        { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
        { to: '/', label: 'Virtual Machines', icon: Server, exact: true },
      ],
    },
    {
      label: 'Resources',
      items: [
        { to: '/networks', label: 'Networks', icon: Network },
        { to: '/disks', label: 'Disks', icon: HardDrive },
        { to: '/media', label: 'Media', icon: Disc3 },
      ],
    },
    {
      label: 'System',
      items: [
        { to: '/jobs', label: 'Activity', icon: ListTodo },
        { to: '/api-docs', label: 'API', icon: Code },
        { to: '/certificates', label: 'Certificates', icon: ShieldCheck },
        ...(admin ? [{ to: '/console', label: 'Console', icon: SquareTerminal }] : []),
      ],
    },
  ]
}

export const mobilePrimary = ['/dashboard', '/', '/networks', '/media']

export function mobileOverflow(admin: boolean): NavItem[] {
  return navigation(admin)
    .flatMap((group) => group.items)
    .filter((item) => !mobilePrimary.includes(item.to))
}

const topLevel = new Set(['/', '/dashboard', '/networks', '/disks', '/media', '/jobs', '/api-docs', '/console', '/more', '/preferences', '/certificates'])

export function isTopLevel(pathname: string): boolean {
  return topLevel.has(pathname)
}
