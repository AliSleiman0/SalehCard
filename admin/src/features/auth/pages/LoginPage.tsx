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
  const verifyTwoFactor = useAuthStore((s) => s.verifyTwoFactor)
  const resendTwoFactor = useAuthStore((s) => s.resendTwoFactor)
  const cancelTwoFactor = useAuthStore((s) => s.cancelTwoFactor)
  const twoFactor = useAuthStore((s) => s.twoFactor)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const from = (location.state as { from?: string })?.from ?? '/'

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const result = await login(email, password)
      if (result === 'ok') {
        navigate(from, { replace: true })
      }
      // result === '2fa' → the store now holds a challenge; the code step renders.
    } catch (ex) {
      setError(ex instanceof Error ? ex.message : 'Login failed. Check your credentials.')
    } finally {
      setBusy(false)
    }
  }

  const submitCode = async (e: React.FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await verifyTwoFactor(code)
      navigate(from, { replace: true })
    } catch (ex) {
      setError(ex instanceof Error ? ex.message : 'Verification failed. Check the code.')
    } finally {
      setBusy(false)
    }
  }

  const resend = async () => {
    setError('')
    setNotice('')
    try {
      await resendTwoFactor()
      setNotice('A new code has been sent.')
    } catch (ex) {
      setError(ex instanceof Error ? ex.message : 'Could not resend the code.')
    }
  }

  const backToPassword = () => {
    cancelTwoFactor()
    setCode('')
    setError('')
    setNotice('')
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
              Flash<span className="grad-text">Cash</span>
            </div>
            <div className="faint" style={{ fontSize: 12 }}>
              {t('login_title')}
            </div>
          </div>
        </div>

        <h2 className="h3" style={{ marginBottom: 4 }}>
          {twoFactor ? 'Two-step verification' : t('login_sub')}
        </h2>

        {twoFactor ? (
          <form onSubmit={submitCode} style={{ marginTop: 18 }}>
            <p className="faint" style={{ fontSize: 12.5, marginBottom: 14, lineHeight: 1.5 }}>
              Enter the 6-digit code sent by SMS to <b>{twoFactor.phoneHint}</b>.
            </p>
            <label className="label">Verification code</label>
            <input
              className="field"
              type="text"
              inputMode="numeric"
              pattern="\d*"
              autoComplete="one-time-code"
              maxLength={6}
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
              placeholder="123456"
              style={{ marginBottom: 18, letterSpacing: 4, fontSize: 18, textAlign: 'center' }}
              autoFocus
            />
            {error && (
              <div
                className="badge badge-soft"
                style={{ display: 'block', marginBottom: 14, color: 'var(--danger, #e5484d)', fontSize: 12.5 }}
              >
                {error}
              </div>
            )}
            {notice && !error && (
              <div className="faint" style={{ marginBottom: 14, fontSize: 12.5 }}>
                {notice}
              </div>
            )}
            <button className="btn btn-primary btn-block" type="submit" disabled={busy || code.length < 6}>
              <Icon name="check" size={16} /> Verify &amp; sign in
            </button>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                marginTop: 14,
                fontSize: 12.5,
              }}
            >
              <button
                type="button"
                className="faint"
                onClick={backToPassword}
                disabled={busy}
                style={{ background: 'none', border: 'none', padding: 0, cursor: 'pointer', font: 'inherit' }}
              >
                ← Back
              </button>
              <button
                type="button"
                className="faint"
                onClick={resend}
                disabled={busy}
                style={{ background: 'none', border: 'none', padding: 0, cursor: 'pointer', font: 'inherit' }}
              >
                Resend code
              </button>
            </div>
          </form>
        ) : (
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
        )}
      </div>
    </div>
  )
}
