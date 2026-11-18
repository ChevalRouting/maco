import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Eye, EyeOff } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

interface LoginFormProps {
  onSubmit: (username: string, password: string) => Promise<void>
  title?: string
  description?: string
  usernameLabel?: string
  passwordLabel?: string
  usernamePlaceholder?: string
  submitLabel?: string
}

export function LoginForm({
  onSubmit,
  title = 'Sign in',
  description = 'Enter your credentials to continue',
  usernameLabel = 'Username',
  passwordLabel = 'Password',
  usernamePlaceholder = 'admin',
  submitLabel = 'Sign in',
}: LoginFormProps) {
  const [username, setUsername] = useState('')
  const [visible, setVisible] = useState(false)
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username || !password) {
      setError('Username and password are required')
      return
    }

    setLoading(true)
    setError(null)
    try {
      await onSubmit(username, password)
    } catch (err) {
      setError((err as Error).message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card
      className="rounded-2xl bg-card/80 backdrop-blur-xl backdrop-saturate-150 [box-shadow:var(--overlay-shadow)]"
    >
      <CardHeader>
        <CardTitle>
          {title}
        </CardTitle>
        <CardDescription>
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={handleSubmit}
          className="space-y-4"
        >
          <div
            className="space-y-2"
          >
            <Label
              htmlFor="username"
            >
              {usernameLabel}
            </Label>
            <Input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={usernamePlaceholder}
              autoComplete="username"
              autoFocus
              required
              disabled={loading}
            />
          </div>
          <div
            className="space-y-2"
          >
            <Label
              htmlFor="password"
            >
              {passwordLabel}
            </Label>
            <div
              className="relative"
            >
              <Input
                id="password"
                type={visible ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                autoComplete="current-password"
                required
                disabled={loading}
                className="pr-10"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="absolute right-0.5 top-0 h-8 w-8 text-muted-foreground hover:bg-transparent hover:text-foreground"
                disabled={loading}
                aria-label={visible ? 'Hide Password' : 'Show Password'}
                aria-pressed={visible}
                onClick={() => setVisible(!visible)}
              >
                {visible ? (
                  <EyeOff
                    aria-hidden="true"
                    className="h-4 w-4"
                  />
                ) : (
                  <Eye
                    aria-hidden="true"
                    className="h-4 w-4"
                  />
                )}
              </Button>
            </div>
          </div>
          {error && (
            <p
              role="alert"
              className="text-sm text-danger"
            >
              {error}
            </p>
          )}
          <Button
            type="submit"
            variant="suggested"
            size="lg"
            className="mt-2 w-full"
            disabled={loading}
          >
            {loading ? 'Signing in…' : submitLabel}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
