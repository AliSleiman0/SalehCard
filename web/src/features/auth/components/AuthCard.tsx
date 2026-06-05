import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { Icon, Logo, Button, Input, useToast } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useLogin } from '@/features/auth/hooks/useLogin'
import { useRegister } from '@/features/auth/hooks/useRegister'

export function AuthCard({ reg }: { reg: boolean }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const toast = useToast()
  const locale = useLocaleStore((s) => s.locale)

  const login = useLogin()
  const register = useRegister()
  const pending = login.isPending || register.isPending

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!email.includes('@')) {
      setError(t('invalid_email'))
      return
    }
    if (password.length < 8) {
      setError(t('password_too_short'))
      return
    }

    const onSuccess = () => navigate('/dashboard')
    const onError = (err: Error) => setError(err.message || t('auth_failed'))

    if (reg) {
      register.mutate({ email, password, locale }, { onSuccess, onError })
    } else {
      login.mutate({ email, password }, { onSuccess, onError })
    }
  }

  return (
    <div
      className="wrap"
      style={{ minHeight: '78vh', display: 'grid', placeItems: 'center', padding: '30px 0' }}
    >
      <div className="card card-pad" style={{ width: '100%', maxWidth: 420, padding: 32 }}>
        <div className="col center" style={{ gap: 14, marginBottom: 22 }}>
          <Logo size={32} />
          <h1 className="h2" style={{ marginTop: 6 }}>
            {reg ? t('create_acct') : t('welcome_back')}
          </h1>
        </div>
        <Button
          variant="ghost"
          size="lg"
          block
          type="button"
          onClick={() => toast(t('coming_soon'))}
        >
          <Icon name="google" size={20} />
          {t('google')}
        </Button>
        <div className="row center" style={{ gap: 14, margin: '20px 0' }}>
          <hr className="divider" style={{ flex: 1 }} />
          <span className="tiny faint">{t('or')}</span>
          <hr className="divider" style={{ flex: 1 }} />
        </div>
        <form className="col" style={{ gap: 14 }} onSubmit={submit}>
          <Input
            label={t('email')}
            type="email"
            placeholder="you@email.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
          />
          <Input
            label={t('password')}
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={reg ? 'new-password' : 'current-password'}
            error={error ?? undefined}
          />
          <Button variant="primary" size="lg" block type="submit" loading={pending}>
            {reg ? t('register') : t('login')}
          </Button>
        </form>
        <div className="row center" style={{ gap: 6, marginTop: 20 }}>
          <span className="small faint">{reg ? t('have_acct') : t('no_acct')}</span>
          <Link
            className="small clickable"
            to={reg ? '/login' : '/register'}
            style={{ fontWeight: 800, color: 'var(--brand-1)' }}
          >
            {reg ? t('login') : t('register')}
          </Link>
        </div>
      </div>
    </div>
  )
}
