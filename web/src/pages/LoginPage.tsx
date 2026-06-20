import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { getAuthConfig, login } from '../api'

export default function LoginPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(
    searchParams.get('error') === 'telegram'
      ? 'Telegram login failed. Try the password instead.'
      : null,
  )
  const [submitting, setSubmitting] = useState(false)
  const [telegramBotUsername, setTelegramBotUsername] = useState('')

  useEffect(() => {
    getAuthConfig()
      .then((cfg) => setTelegramBotUsername(cfg.telegramBotUsername))
      .catch(() => setTelegramBotUsername(''))
  }, [])

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      await login(password)
      navigate(searchParams.get('next') || '/')
    } catch (err) {
      setError(String(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="login-page">
      <h1>Amethyst</h1>

      {telegramBotUsername && <TelegramLoginWidget botUsername={telegramBotUsername} />}

      <form className="login-form" onSubmit={onSubmit}>
        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoFocus
          required
        />
        <button type="submit" disabled={submitting}>
          {submitting ? 'Logging in…' : 'Log in'}
        </button>
      </form>

      {error && (
        <p role="alert" className="login-error">
          {error}
        </p>
      )}
    </div>
  )
}

// Telegram's widget script replaces its own <script> tag with an iframe.
// data-auth-url makes it redirect the browser straight to
// GET /api/auth/telegram/callback (signature-verified server-side) instead
// of needing a JS callback here.
function TelegramLoginWidget({ botUsername }: { botUsername: string }) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return
    container.innerHTML = ''

    const script = document.createElement('script')
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.async = true
    script.setAttribute('data-telegram-login', botUsername)
    script.setAttribute('data-size', 'large')
    script.setAttribute(
      'data-auth-url',
      `${window.location.origin}/api/auth/telegram/callback`,
    )
    script.setAttribute('data-request-access', 'write')
    container.appendChild(script)
  }, [botUsername])

  return <div ref={containerRef} className="login-telegram" />
}
