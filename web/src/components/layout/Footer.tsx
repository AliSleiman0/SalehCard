import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Logo, Badge } from '@/components'
import { useLocaleStore } from '@/stores/locale'
import { useCategories } from '@/features/catalog/hooks/useCategories'
import { adaptRootCategory } from '@/features/catalog/lib/adaptCategory'

export function Footer() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const locale = useLocaleStore((s) => s.locale)
  const cats = (useCategories({ depth: 0 }).data?.data ?? [])
    .map((c) => adaptRootCategory(c, locale))
    .sort((a, b) => a.order - b.order)

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
            {cats.slice(0, 4).map((c) => (
              <a
                key={c.key}
                className="small muted clickable"
                onClick={() => navigate('/category/' + c.key)}
              >
                {c.name}
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
            {/* The "agent portal" link was removed: reseller onboarding is
                admin-managed, and /reseller is role-gated (RequireReseller) —
                for a non-reseller the link would just bounce to /dashboard. */}
          </div>
        </div>
      </div>
    </footer>
  )
}
