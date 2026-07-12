import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Icon, Panel, Button, LoadingSpinner } from '@/components'
import type { PaymentIntent } from '@/types'
import { usePaymentIntent } from '../hooks/usePaymentIntent'
import { useCountdown } from '../hooks/useCountdown'
import { formatUsdtAmount } from '../lib/usdt'
import { QrCode } from '../components/QrCode'

function CopyRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  const [copied, setCopied] = useState(false)
  const copy = () => {
    navigator.clipboard?.writeText(value).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1600)
    })
  }
  return (
    <div className="col" style={{ gap: 4 }}>
      <span className="tiny faint">{label}</span>
      <div className="row between" style={{ gap: 8 }}>
        <span
          style={{
            fontFamily: mono ? 'var(--font-mono, monospace)' : undefined,
            fontWeight: 700,
            wordBreak: 'break-all',
            fontSize: mono ? 13 : 18,
          }}
        >
          {value}
        </span>
        <button className="icon-btn" style={{ width: 34, height: 34, flex: 'none' }} onClick={copy}>
          <Icon name={copied ? 'check' : 'copy'} size={15} />
        </button>
      </div>
    </div>
  )
}

export default function UsdtDepositPage() {
  const { t } = useTranslation()
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const location = useLocation()
  const seed = (location.state as { intent?: PaymentIntent } | null)?.intent

  const { data: intent, isLoading } = usePaymentIntent(id, seed)
  const { label: countdown, done: countdownDone } = useCountdown(intent?.expiresAt)
  const navigated = useRef(false)

  useEffect(() => {
    if (!intent || navigated.current) return
    if (intent.status === 'confirmed') {
      navigated.current = true
      void qc.invalidateQueries({ queryKey: ['wallet'] })
      void qc.invalidateQueries({ queryKey: ['topup-requests'] })
      void qc.invalidateQueries({ queryKey: ['orders'] })
      if (intent.purpose === 'order' && intent.orderId) navigate('/orders/' + intent.orderId)
      else navigate('/wallet')
    }
  }, [intent, navigate, qc])

  if (isLoading || !intent) {
    return (
      <div className="wrap" style={{ padding: '40px 0' }}>
        <LoadingSpinner />
      </div>
    )
  }

  const expired = intent.status === 'expired' || (countdownDone && intent.status !== 'confirmed')
  const confirming = intent.status === 'confirming'
  const net = intent.network.toUpperCase()

  return (
    <div className="wrap" style={{ minHeight: '70vh', display: 'grid', placeItems: 'center', padding: '30px 0' }}>
      <Panel style={{ width: '100%', maxWidth: 460, display: 'flex', flexDirection: 'column', gap: 16, padding: 28 }}>
        {expired ? (
          <div className="col center" style={{ gap: 12, textAlign: 'center', padding: '20px 0' }}>
            <span className="mi" style={{ width: 52, height: 52, background: 'var(--danger)' }}>
              <Icon name="close" size={24} />
            </span>
            <h1 className="h2">{t('usdt_expired_title')}</h1>
            <p className="muted">{t('usdt_expired_body')}</p>
            <Button variant="primary" size="lg" block onClick={() => navigate('/wallet')}>
              {t('back_to_wallet')}
            </Button>
          </div>
        ) : (
          <>
            <div className="col center" style={{ gap: 6, textAlign: 'center' }}>
              <h1 className="h2">{t('usdt_pay_title')}</h1>
              <p className="muted small">{t('usdt_pay_sub')}</p>
            </div>

            {confirming && (
              <div
                className="row"
                style={{ gap: 8, justifyContent: 'center', color: 'var(--ok)', fontWeight: 700 }}
              >
                <Icon name="check" size={16} /> {t('usdt_confirming')}
              </div>
            )}

            <div className="col center">
              <QrCode value={intent.address} />
            </div>

            <CopyRow label={t('usdt_send_exactly')} value={`${formatUsdtAmount(intent.amountUsd)} USDT`} />
            <CopyRow label={t('usdt_to_address', { net })} value={intent.address} mono />

            <div
              className="card"
              style={{
                padding: '10px 14px',
                background: 'rgba(229,72,77,.1)',
                border: '1px solid rgba(229,72,77,.4)',
                color: 'var(--danger)',
              }}
            >
              <span className="tiny" style={{ fontWeight: 700 }}>
                {t('usdt_network_warning', { net })}
              </span>
            </div>

            <div className="row between">
              <span className="row" style={{ gap: 8, color: 'var(--text-dim)' }}>
                <LoadingSpinner size={16} />
                <span className="small" style={{ fontWeight: 700 }}>
                  {t('usdt_waiting')}
                </span>
              </span>
              <span className="num" style={{ fontWeight: 800 }}>
                {countdown}
              </span>
            </div>
          </>
        )}
      </Panel>
    </div>
  )
}
