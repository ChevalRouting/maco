import { PageHeader, PreferencesGroup, Row, Segmented, useTheme } from 'maco-ui'
import { useSession } from '../hooks/useSession'
import { APPEARANCE_OPTIONS } from '../components/shell/MainMenu'
import { PasswordForm } from '../components/PasswordForm'
import { SSHKeys } from '../components/SSHKeys'

const THEME_OPTIONS = APPEARANCE_OPTIONS.map(({ value, label }) => ({ value, label }))

export default function Preferences() {
  const session = useSession()
  const { preference, setPreference } = useTheme()

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader title="Preferences" />

      <PreferencesGroup title="Account">
        <Row title="Username">
          <span className="font-mono text-sm">{session.user?.username || 'Unknown'}</span>
        </Row>
        <Row title="Role">
          <span className="text-sm">{session.admin ? 'Administrator' : 'Viewer'}</span>
        </Row>
      </PreferencesGroup>

      <PreferencesGroup title="Appearance">
        <Row title="Style">
          <Segmented value={preference} onChange={setPreference} options={THEME_OPTIONS} />
        </Row>
      </PreferencesGroup>

      <PasswordForm />
      <SSHKeys />
    </div>
  )
}
