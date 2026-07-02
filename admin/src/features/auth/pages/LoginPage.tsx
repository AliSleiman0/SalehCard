import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useLocation } from 'react-router-dom'
import { Icon } from '@/components'
import { useAuthStore } from '@/stores/auth'

export default function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const login = useAuthStore((s) => s.login)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const from = (location.state as { from?: string })?.from ?? '/'

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await login(email, password)
      navigate(from, { replace: true })
    } catch (ex) {
      setError(ex instanceof Error ? ex.message : 'Login failed. Check your credentials.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        padding: 20,
        background: 'var(--bg-grad)',
      }}
    >
      <div className="card card-pad" style={{ width: '100%', maxWidth: 400 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 11, marginBottom: 20 }}>
          <div className="sb-mark" style={{ width: 40, height: 40, borderRadius: 12 }}>
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
              <path d="M16 5H10a3 3 0 0 0 0 6h4a3 3 0 0 1 0 6H7" stroke="#fff" strokeWidth="2.6" strokeLinecap="round" />
              <circle cx="18.5" cy="6" r="1.7" fill="#22e3c8" />
            </svg>
          </div>
          <div>
            <div style={{ fontFamily: 'var(--font-display)', fontSize: 22, lineHeight: 1 }}>
              Saleh<span className="grad-text">Card</span>
            </div>
            <div className="faint" style={{ fontSize: 12 }}>
              {t('login_title')}
            </div>
          </div>
        </div>

        <h2 className="h3" style={{ marginBottom: 4 }}>
          {t('login_sub')}
        </h2>

        <form onSubmit={submit} style={{ marginTop: 18 }}>
          <label className="label">{t('login_email')}</label>
          <input
            className="field"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="admin@salehcard.co"
            style={{ marginBottom: 14 }}
            autoFocus
          />
          <label className="label">{t('login_password')}</label>
          <input
            className="field"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            style={{ marginBottom: 18 }}
          />
          {error && (
            <div
              className="badge badge-soft"
              style={{ display: 'block', marginBottom: 14, color: 'var(--danger, #e5484d)', fontSize: 12.5 }}
            >
              {error}
            </div>
          )}
          <button className="btn btn-primary btn-block" type="submit" disabled={busy}>
            <Icon name="logout" size={16} /> {t('login_submit')}
          </button>
        </form>
      </div>
    </div>
  )
}
