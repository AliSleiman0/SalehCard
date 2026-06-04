import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { Icon, Logo, Button, Input } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { useWalletStore } from '@/stores/wallet'
import { DEMO } from '@/lib/mock/demo'

export function AuthCard({ reg }: { reg: boolean }) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const go = () => {
    useAuthStore.getState().setUser({
      id: 'demo-user',
      email: DEMO.user.email,
      role: 'customer',
      locale: 'en',
      savedPlayerIds: [],
      walletBalance: useWalletStore.getState().balance,
      loyaltyPoints: DEMO.user.loyalty,
      createdAt: '',
      updatedAt: '',
    })
    navigate('/dashboard')
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
        <Button variant="ghost" size="lg" block onClick={go}>
          <Icon name="google" size={20} />
          {t('google')}
        </Button>
        <div className="row center" style={{ gap: 14, margin: '20px 0' }}>
          <hr className="divider" style={{ flex: 1 }} />
          <span className="tiny faint">{t('or')}</span>
          <hr className="divider" style={{ flex: 1 }} />
        </div>
        <div className="col" style={{ gap: 14 }}>
          {reg && <Input label="Name" placeholder="Yusuf Demir" />}
          <Input label={t('email')} placeholder="you@email.com" />
          <Input label={t('password')} type="password" placeholder="••••••••" />
          <Button variant="primary" size="lg" block onClick={go}>
            {reg ? t('register') : t('login')}
          </Button>
        </div>
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
