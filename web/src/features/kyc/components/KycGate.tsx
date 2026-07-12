import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Panel, Button } from '@/components'
import type { KycStatus } from '../types'

// KycGate is the checkout blocking panel shown when the customer isn't verified.
// Pending → informational; unverified/rejected → CTA to the KYC flow.
export function KycGate({ status }: { status: KycStatus }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const pending = status === 'pending'

  return (
    <Panel style={{ display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'flex-start' }}>
      <span className="mi" style={{ background: 'var(--grad)' }}>
        <Icon name="shield" size={18} />
      </span>
      <h3 className="h3">{pending ? t('kyc_pending_title') : t('kyc_gate_title')}</h3>
      <p className="small muted" style={{ margin: 0 }}>
        {pending ? t('kyc_pending_body') : t('kyc_gate_body')}
      </p>
      {!pending && (
        <Button variant="primary" size="lg" onClick={() => navigate('/kyc')}>
          <Icon name="shield" size={16} />
          {t('kyc_verify_now')}
        </Button>
      )}
    </Panel>
  )
}
