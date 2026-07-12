import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Icon, Logo, Button, Segmented, useToast } from '@/components'
import { LoginCard } from './LoginCard'
import { SignupCard } from './SignupCard'

// AuthCard is the shared auth shell: logo, Google button, a Phone|Email tab, the
// form body (LoginCard/SignupCard), and the login↔register switch link.
export function AuthCard({ reg }: { reg: boolean }) {
  const { t } = useTranslation()
  const toast = useToast()
  const [tab, setTab] = useState<'phone' | 'email'>('phone')

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

        <Button variant="ghost" size="lg" block type="button" onClick={() => toast(t('coming_soon'))}>
          <Icon name="google" size={20} />
          {t('google')}
        </Button>

        <div className="row center" style={{ gap: 14, margin: '20px 0' }}>
          <hr className="divider" style={{ flex: 1 }} />
          <span className="tiny faint">{t('or')}</span>
          <hr className="divider" style={{ flex: 1 }} />
        </div>

        <div style={{ marginBottom: 16 }}>
          <Segmented
            options={[
              { value: 'phone', label: t('phone') },
              { value: 'email', label: t('email') },
            ]}
            value={tab}
            onChange={setTab}
          />
        </div>

        {reg ? <SignupCard tab={tab} /> : <LoginCard tab={tab} />}

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
