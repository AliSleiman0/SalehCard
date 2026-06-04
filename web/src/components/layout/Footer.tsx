import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Logo, Badge } from '@/components'
import { CATEGORIES } from '@/lib/mock/demo'

export function Footer() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <footer className="footer desktop-only">
      <div className="wrap row between wrap-gap" style={{ alignItems: 'flex-start', gap: 30 }}>
        <div style={{ maxWidth: 280 }}>
          <Logo size={26} />
          <p className="small muted" style={{ marginTop: 14 }}>
            {t('hero_sub')}
          </p>
          <div className="row" style={{ gap: 8, marginTop: 16 }}>
            <Badge variant="instant">
              <Icon name="bolt" size={12} />
              {t('trust_instant')}
            </Badge>
            <Badge variant="secure">
              <Icon name="shield" size={12} />
              {t('trust_secure')}
            </Badge>
          </div>
        </div>
        <div className="row wrap-gap" style={{ gap: 40, alignItems: 'flex-start' }}>
          <div className="col" style={{ gap: 10 }}>
            <span className="eyebrow">Store</span>
            {CATEGORIES.slice(0, 4).map((c) => (
              <a
                key={c.id}
                className="small muted clickable"
                onClick={() => navigate('/category/' + c.id)}
              >
                {t(c.key)}
              </a>
            ))}
          </div>
          <div className="col" style={{ gap: 10 }}>
            <span className="eyebrow">Account</span>
            <a className="small muted clickable" onClick={() => navigate('/wallet')}>
              {t('nav_wallet')}
            </a>
            <a className="small muted clickable" onClick={() => navigate('/orders')}>
              {t('order_history')}
            </a>
            <a className="small muted clickable" onClick={() => navigate('/reseller')}>
              {t('become_agent')}
            </a>
          </div>
        </div>
      </div>
    </footer>
  )
}
