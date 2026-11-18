import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { getMe, isAdmin, errorMessage, type CurrentUser } from '../api'
interface SessionState { user: CurrentUser | null; admin: boolean; loading: boolean; error: string; refresh: () => void }
const SessionContext = createContext<SessionState>({ user: null, admin: false, loading: true, error: '', refresh: () => {} })
interface SessionProviderProps { children: ReactNode }
export function SessionProvider({ children }: SessionProviderProps) {
  const [revision, setRevision] = useState(0)
  const [state, setState] = useState<Omit<SessionState, 'refresh'>>({ user: null, admin: false, loading: true, error: '' })
  useEffect(() => {
    let active = true
    setState(current => ({ ...current, loading: true, error: '' }))
    getMe().then(user => { if (active) setState({ user, admin: isAdmin(user), loading: false, error: '' }) }).catch(error => { if (active) setState({ user: null, admin: false, loading: false, error: errorMessage(error) }) })
    return () => { active = false }
  }, [revision])
  return <SessionContext.Provider value={{ ...state, refresh: () => setRevision(value => value + 1) }}>{children}</SessionContext.Provider>
}
export function useSession() { return useContext(SessionContext) }
