import { useNavigate, useSearchParams } from 'react-router-dom'
import { LoginScreen, LoginForm, NoticeBanner } from 'maco-ui'
import { login } from '../api'
import { safeReturnPath } from '../ux'
import { Logo } from '../Logo'

export default function Login() {
  const navigate = useNavigate()
  const [params] = useSearchParams()

  async function submit(username: string, password: string) {
    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur()
    }
    await login(username, password)
    navigate(safeReturnPath(params.get('returnTo')), { replace: true })
  }

  return (
    <LoginScreen
      title="maco"
      subtitle="macOS virtual machine manager"
      logo={Logo}
    >
      {params.has('expired') && (
        <div className="relative z-10 -mb-3 overflow-hidden rounded-t-xl bg-background">
          <NoticeBanner intent="info">
            Session expired
          </NoticeBanner>
        </div>
      )}
      <LoginForm onSubmit={submit} />
    </LoginScreen>
  )
}
