import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Panel, Button, LoadingSpinner } from '@/components'
import { useKycProfile } from '../hooks/useKycProfile'

// KycPage is the standalone verification status screen (reachable pre-purchase
// from the home banner and the account menu).
export default function KycPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading } = useKycProfile()

  if (isLoading || !data) {
    return (
      <div className="wrap" style={{ padding: '40px 0' }}>
        <LoadingSpinner />
      </div>
    )
  }

  return (
    <div className="wrap" style={{ minHeight: '70vh', display: 'grid', placeItems: 'center', padding: '30px 0' }}>
      <Panel style={{ width: '100%', maxWidth: 460, display: 'flex', flexDirection: 'column', gap: 14, alignItems: 'center', textAlign: 'center', padding: 32 }}>
        <span
          className="mi"
          style={{
            width: 56,
            height: 56,
            background: data.status === 'verified' ? 'var(--ok)' : 'var(--grad)',
          }}
        >
          <Icon name={data.status === 'verified' ? 'check' : 'shield'} size={26} />
        </span>

        {data.status === 'verified' && (
          <>
            <h1 className="h2">{t('kyc_verified_title')}</h1>
            <p className="muted">{t('kyc_verified_body')}</p>
            <Button variant="primary" size="lg" block onClick={() => navigate('/')}>
              {t('continue_shop')}
            </Button>
          </>
        )}

        {data.status === 'pending' && (
          <>
            <h1 className="h2">{t('kyc_pending_title')}</h1>
            <p className="muted">{t('kyc_pending_body')}</p>
            <Button variant="ghost" size="lg" block onClick={() => navigate('/')}>
              {t('continue_shop')}
            </Button>
          </>
        )}

        {(data.status === 'unverified' || data.status === 'rejected') && (
          <>
            <h1 className="h2">
              {data.status === 'rejected' ? t('kyc_rejected_title') : t('kyc_gate_title')}
            </h1>
            <p className="muted">
              {data.status === 'rejected' && data.rejectionReason
                ? data.rejectionReason
                : t('kyc_gate_body')}
            </p>
            <Button variant="primary" size="lg" block onClick={() => navigate('/kyc/submit')}>
              <Icon name="shield" size={16} />
              {data.status === 'rejected' ? t('kyc_resubmit') : t('kyc_start')}
            </Button>
          </>
        )}
      </Panel>
    </div>
  )
}
