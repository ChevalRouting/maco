import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader, PreferencesGroup, Row, Segmented, useTheme } from 'maco-ui'
import { Info, LogOut, SlidersHorizontal } from 'lucide-react'
import { mobileOverflow } from '../lib/navigation'
import { useSession } from '../hooks/useSession'
import { AboutDialog } from '../components/shell/AboutDialog'
import { APPEARANCE_OPTIONS } from '../components/shell/MainMenu'
import { signOut } from '../components/shell/signOut'

const THEME_OPTIONS = APPEARANCE_OPTIONS.map(({ value, label }) => ({ value, label }))

export default function More() {
  const navigate = useNavigate()
  const { admin } = useSession()
  const { preference, setPreference } = useTheme()
  const [aboutOpen, setAboutOpen] = useState(false)

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader title="More" />

      <PreferencesGroup>
        {mobileOverflow(admin).map((item) => (
          <Row key={item.to} title={item.label} prefix={<item.icon className="h-5 w-5" />} onClick={() => navigate(item.to)} />
        ))}
      </PreferencesGroup>

      <PreferencesGroup>
        <Row title="Preferences" prefix={<SlidersHorizontal className="h-5 w-5" />} onClick={() => navigate('/preferences')} />
        <Row title="Appearance">
          <Segmented value={preference} onChange={setPreference} options={THEME_OPTIONS} />
        </Row>
      </PreferencesGroup>

      <PreferencesGroup>
        <Row title="About maco" prefix={<Info className="h-5 w-5" />} onClick={() => setAboutOpen(true)} />
        <Row title="Sign Out" prefix={<LogOut className="h-5 w-5" />} onClick={signOut} />
      </PreferencesGroup>

      <AboutDialog open={aboutOpen} onClose={() => setAboutOpen(false)} />
    </div>
  )
}
