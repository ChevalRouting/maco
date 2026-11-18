import { createContext, createElement, useContext, useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

export type ThemePreference = Theme | 'system'

interface ThemeContextValue {
  theme: Theme
  preference: ThemePreference
  setPreference: (preference: ThemePreference) => void
  toggle: () => void
}

export const ThemeContext = createContext<ThemeContextValue>({
  theme: 'dark',
  preference: 'system',
  setPreference: () => {},
  toggle: () => {},
})

const DARK_QUERY = '(prefers-color-scheme: dark)'

function isPreference(value: string | null): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

function getInitialPreference(storageKey: string, defaultPreference: ThemePreference): ThemePreference {
  const stored = localStorage.getItem(storageKey)
  if (isPreference(stored)) {
    return stored
  }
  return defaultPreference
}

function systemTheme(): Theme {
  return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light'
}

export function useTheme() {
  return useContext(ThemeContext)
}

export function useThemeState(
  storageKey = 'maco-ui-theme',
  defaultPreference: ThemePreference = 'system',
): ThemeContextValue {
  const [preference, setPreference] = useState<ThemePreference>(() => getInitialPreference(storageKey, defaultPreference))
  const [system, setSystem] = useState<Theme>(systemTheme)
  const theme = preference === 'system' ? system : preference

  useEffect(() => {
    const query = window.matchMedia(DARK_QUERY)
    const handleChange = () => setSystem(systemTheme())
    query.addEventListener('change', handleChange)
    return () => query.removeEventListener('change', handleChange)
  }, [])

  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('dark', theme === 'dark')
    root.style.colorScheme = theme
  }, [theme])

  useEffect(() => {
    localStorage.setItem(storageKey, preference)
  }, [storageKey, preference])

  const toggle = () => setPreference(theme === 'dark' ? 'light' : 'dark')

  return { theme, preference, setPreference, toggle }
}

export function ThemeProvider({
  children,
  storageKey,
  defaultTheme,
}: {
  children: React.ReactNode
  storageKey?: string
  defaultTheme?: ThemePreference
}) {
  const value = useThemeState(storageKey, defaultTheme)
  return createElement(ThemeContext.Provider, { value }, children)
}
