import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon } from '@/components'
import { useAuthStore } from '@/stores/auth'
import { useKycProfile } from '../hooks/useKycProfile'

// KycBanner nudges unverified/rejected users to verify (and shows a calm
// "under review" note when pending). Renders nothing when signed out, loading,
// errored, or already verified. Port of the app's kyc_banner.
export function KycBanner() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const { data } = useKycProfile()

  if (!isAuthenticated || !data || data.status === 'verified') return null

  if (data.status === 'pending') {
    return (
      <div
        className="card"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '12px 16px',
          marginBottom: 16,
          background: 'rgba(255,176,46,.12)',
          border: '1px solid rgba(255,176,46,.4)',
        }}
      >
        <Icon name="shield" size={18} />
        <span className="small" style={{ fontWeight: 700 }}>
          {t('kyc_pending_banner')}
        </span>
      </div>
    )
  }

  return (
    <div
      className="card clickable hover-pop"
      onClick={() => navigate('/kyc')}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '12px 16px',
        marginBottom: 16,
        background: 'var(--grad-soft)',
        border: '1px solid var(--border-strong)',
      }}
    >
      <span className="mi" style={{ background: 'var(--grad)', flex: 'none' }}>
        <Icon name="shield" size={18} />
      </span>
      <div className="col" style={{ gap: 1, flex: 1 }}>
        <span style={{ fontWeight: 800 }}>{t('kyc_verify_title')}</span>
        <span className="tiny faint">{t('kyc_verify_sub')}</span>
      </div>
      <Icon name="chevron" size={18} />
    </div>
  )
}
