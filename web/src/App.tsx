import { Routes, Route, Navigate, Outlet, useLocation } from 'react-router-dom'
import { AppShell, Button, MobileNav, NoticeBanner, Spinner, Toaster, type MobileTab } from 'maco-ui'
import { MoreHorizontal } from 'lucide-react'
import { getToken } from './api'
import { Logo } from './Logo'
import Login from './pages/Login'
import VMs from './pages/VMs'
import Dashboard from './pages/Dashboard'
import Networks from './pages/Networks'
import Disks from './pages/Disks'
import Media from './pages/Media'
import VMDetails from './pages/VMDetails'
import Preferences from './pages/Preferences'
import Certificates from './pages/Certificates'
import More from './pages/More'
import Jobs from './pages/Jobs'
import JobDetails from './pages/JobDetails'
import Api from './pages/Api'
import Console from './pages/Console'
import { JobsProvider } from './components/JobsProvider'
import { JobsPanel } from './components/JobsPanel'
import { MobileJobsIndicator } from './components/MobileJobsIndicator'
import { MainMenu } from './components/shell/MainMenu'
import { MobileHeader } from './components/shell/MobileHeader'
import { SessionProvider, useSession } from './hooks/useSession'
import { useStandaloneViewport } from './hooks/useStandaloneViewport'
import { useCompact } from './hooks/useNarrow'
import { isTopLevel, mobilePrimary, navigation } from './lib/navigation'
import { useScrollRecovery } from './lib/useScrollRecovery'

const MOBILE_LABELS: Record<string, string> = {
  '/': 'VMs',
}

function mobileTabs(admin: boolean): MobileTab[] {
  const items = navigation(admin).flatMap((group) => group.items)
  const tabs: MobileTab[] = mobilePrimary.flatMap((path) => {
    const item = items.find((entry) => entry.to === path)
    return item ? [{ key: item.to, label: MOBILE_LABELS[item.to] ?? item.label, icon: item.icon, to: item.to, exact: item.exact }] : []
  })
  tabs.push({ key: '/more', label: 'More', icon: MoreHorizontal, to: '/more' })
  return tabs
}

interface SessionShellProps {
  children: React.ReactNode
}

function SessionContent({ children }: SessionShellProps) {
  const session = useSession()

  if (session.loading) return <Spinner />
  if (session.error) {
    return (
      <NoticeBanner intent="danger">
        <div className="flex flex-wrap items-center gap-3">
          <p className="min-w-0 flex-1">Account unavailable: {session.error}</p>
          <Button onClick={session.refresh}>Retry</Button>
        </div>
      </NoticeBanner>
    )
  }

  return <>{children}</>
}

function SessionShell({ children }: SessionShellProps) {
  const session = useSession()
  const location = useLocation()
  useScrollRecovery(location.pathname)

  return (
    <AppShell
      title="maco"
      logo={Logo}
      nav={navigation(session.admin)}
      showThemeToggle={false}
      showMobileHeader={false}
      isRootPath={isTopLevel}
      banners={<MobileHeader />}
      sidebarTop={({ collapsed }) => <MainMenu collapsed={collapsed} />}
      mobileNavigation={
        <>
          <div className="hidden md:block">
            <JobsPanel />
          </div>
          <MobileNav className="mobile-nav" tabs={mobileTabs(session.admin)} subNav={false} />
        </>
      }
      mobileOverlay={<MobileJobsIndicator />}
    >
      <SessionContent>{children}</SessionContent>
    </AppShell>
  )
}

function Shell() {
  const location = useLocation()
  const compact = useCompact()
  if (!getToken()) {
    return <Navigate to={`/login?returnTo=${encodeURIComponent(location.pathname + location.search)}`} replace />
  }

  return (
    <SessionProvider>
      <JobsProvider>
        <div className="app-frame">
          <SessionShell>
            <Outlet />
          </SessionShell>
        </div>
        <Toaster position={compact ? 'top-center' : 'bottom-right'} />
      </JobsProvider>
    </SessionProvider>
  )
}

export default function App() {
  useStandaloneViewport()

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route element={<Shell />}>
        <Route path="/" element={<VMs />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/networks" element={<Networks />} />
        <Route path="/disks" element={<Disks />} />
        <Route path="/media" element={<Media />} />
        <Route path="/vms/:id" element={<VMDetails />} />
        <Route path="/preferences" element={<Preferences />} />
        <Route path="/certificates" element={<Certificates />} />
        <Route path="/settings" element={<Navigate to="/certificates" replace />} />
        <Route path="/profile" element={<Navigate to="/preferences" replace />} />
        <Route path="/more" element={<More />} />
        <Route path="/api-docs" element={<Api />} />
        <Route path="/console" element={<Console />} />
        <Route path="/jobs" element={<Jobs />} />
        <Route path="/jobs/:id" element={<JobDetails />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
