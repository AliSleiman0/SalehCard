import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, Button, useToast, LoadingSpinner } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { NetworkChips } from '@/features/payments/components/NetworkChips'
import { usePaymentConfig } from '@/features/payments/hooks/usePaymentConfig'
import { useCreateTopUpIntent } from '@/features/payments/hooks/useCreateTopUpIntent'
import { useCurrencyStore } from '@/stores/currency'
import { fmtPrice } from '@/lib/utils'
import { useWallet } from '../hooks/useWallet'
import { useTopUp, useTopUpRequests } from '../hooks/useTopUp'
import type { TopUpChannel } from '../api/wallet'
import type { WalletTransaction } from '@/types'

const PRESETS = [25, 50, 100, 250]

// Out-of-band payment channels for top-up requests (credited after an admin
// confirms receipt — there is no instant/self-serve card gateway).
const CHANNELS: { k: TopUpChannel; label: string; badge: string; c: string }[] = [
  { k: 'whish', label: 'Whish', badge: 'W', c: '#e0195c' },
  { k: 'omt', label: 'OMT', badge: 'O', c: '#f39200' },
  { k: 'cash', label: 'Cash', badge: '$', c: '#4a4f5c' },
  { k: 'usdt', label: 'USDT', badge: '₮', c: '#26a17b' },
]

const REQ_STATUS_COLOR: Record<string, string> = {
  pending: 'var(--warn, #f5a623)',
  approved: 'var(--ok)',
  rejected: 'var(--danger, #e5484d)',
}

// txLabel renders a human label for a ledger row from its type.
function txLabel(tx: WalletTransaction, t: (k: string) => string): string {
  switch (tx.type) {
    case 'topup':
      return t('topup')
    case 'purchase':
      return t('checkout')
    case 'refund':
      return t('refunded')
    default:
      return tx.type
  }
}

function fmtTxDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export default function WalletPage() {
  const { t } = useTranslation()
  const toast = useToast()
  const navigate = useNavigate()
  const cur = useCurrencyStore((s) => s.currency)
  const walletQuery = useWallet()
  const topUp = useTopUp()
  const requestsQuery = useTopUpRequests()
  const paymentConfig = usePaymentConfig()
  const createIntent = useCreateTopUpIntent()
  const keyRef = useRef<string | null>(null)
  const [amt, setAmt] = useState(50)
  const [via, setVia] = useState<TopUpChannel>('whish')
  const [usdtNetwork, setUsdtNetwork] = useState('')

  const balance = walletQuery.data?.balance ?? 0
  const txs = walletQuery.data?.transactions ?? []
  const requests = requestsQuery.data ?? []
  const usdtEnabled = paymentConfig.data?.usdtEnabled === true
  const networks = paymentConfig.data?.networks ?? []
  // USDT chosen + on-chain enabled → the instant deposit flow (not a manual request).
  const usdtInstant = via === 'usdt' && usdtEnabled

  const onTopUp = (): void => {
    if (topUp.isPending || createIntent.isPending) return

    if (usdtInstant) {
      const network = usdtNetwork || paymentConfig.data?.network || networks[0]
      keyRef.current = crypto.randomUUID()
      createIntent.mutate(
        { amount: amt, network, idempotencyKey: keyRef.current },
        {
          onSuccess: (intent) =>
            navigate('/payments/usdt-deposit/' + intent.id, { state: { intent } }),
          onError: (err) => toast(err.message || t('failed_title'), 'user'),
        },
      )
      return
    }

    topUp.mutate(
      { amount: amt, channel: via },
      {
        onSuccess: () =>
          toast('Request submitted — your wallet is credited once we confirm your payment.', 'wallet'),
        onError: (err) => toast(err.message || 'Top-up request failed', 'user'),
      },
    )
  }

  return (
    <div className="wrap" style={{ padding: '26px 0 50px' }}>
      <h1 className="h1" style={{ marginBottom: 24 }}>
        {t('your_wallet')}
      </h1>
      <div className="cols-acct">
        <AcctSidebar active="wallet" />
        <div className="col" style={{ gap: 24 }}>
          <div className="cols2">
            <div className="col" style={{ gap: 20 }}>
              <div className="card bigbal card-pad" style={{ padding: 28 }}>
                <span className="eyebrow" style={{ color: 'rgba(255,255,255,.8)' }}>
                  {t('current_balance')}
                </span>
                <div className="display-xl num" style={{ color: '#fff', margin: '6px 0' }}>
                  {fmtPrice(balance, cur)}
                </div>
              </div>
              <div className="panel card-pad">
                <h3 className="h3" style={{ marginBottom: 14 }}>
                  {t('topup_wallet')}
                </h3>
                <div className="label">{t('amount')}</div>
                <div className="row wrap-gap" style={{ gap: 8, marginBottom: 14 }}>
                  {PRESETS.map((p) => (
                    <Button
                      key={p}
                      variant={amt === p ? 'primary' : 'ghost'}
                      size="sm"
                      onClick={() => setAmt(p)}
                    >
                      {fmtPrice(p, cur)}
                    </Button>
                  ))}
                </div>
                <div className="label">{t('topup_via')}</div>
                <div className="row wrap-gap" style={{ gap: 10, marginBottom: 12 }}>
                  {CHANNELS.map((ch) => (
                    <div
                      key={ch.k}
                      className={'method' + (via === ch.k ? ' on' : '')}
                      style={{ flex: 1, padding: 12, minWidth: 90 }}
                      onClick={() => setVia(ch.k)}
                    >
                      <span className="mi" style={{ background: ch.c }}>
                        {ch.badge}
                      </span>
                      <span style={{ fontWeight: 700 }}>{ch.label}</span>
                    </div>
                  ))}
                </div>
                {usdtInstant && networks.length > 1 && (
                  <div style={{ marginBottom: 12 }}>
                    <div className="label">{t('usdt_network')}</div>
                    <NetworkChips
                      networks={networks}
                      value={usdtNetwork || paymentConfig.data?.network || networks[0]}
                      onChange={setUsdtNetwork}
                    />
                  </div>
                )}
                <p className="tiny muted" style={{ marginBottom: 14 }}>
                  {usdtInstant
                    ? t('usdt_topup_note')
                    : 'Top-ups are credited after we confirm your payment — usually within a few minutes during business hours.'}
                </p>
                <Button
                  variant="primary"
                  size="lg"
                  block
                  disabled={topUp.isPending || createIntent.isPending}
                  onClick={onTopUp}
                >
                  {topUp.isPending || createIntent.isPending
                    ? t('processing')
                    : usdtInstant
                      ? `${t('pay_with_usdt')} · ${fmtPrice(amt, cur)}`
                      : `Request top-up · ${fmtPrice(amt, cur)}`}
                </Button>
                {requests.length > 0 && (
                  <div style={{ marginTop: 18 }}>
                    <div className="label">My top-up requests</div>
                    {requests.slice(0, 5).map((r) => (
                      <div className="lrow" key={r.id}>
                        <div className="col" style={{ gap: 1, flex: 1 }}>
                          <span style={{ fontWeight: 700, fontSize: 14 }}>
                            {fmtPrice(r.amount, cur)}{' '}
                            <span className="tiny faint">· {r.channel.toUpperCase()}</span>
                          </span>
                          {r.status === 'rejected' && r.decisionReason && (
                            <span className="tiny" style={{ color: 'var(--danger, #e5484d)' }}>
                              {r.decisionReason}
                            </span>
                          )}
                        </div>
                        <span
                          className="tiny"
                          style={{ fontWeight: 800, color: REQ_STATUS_COLOR[r.status] ?? 'var(--text-dim)' }}
                        >
                          {r.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className="panel card-pad">
              <h3 className="h3" style={{ marginBottom: 8 }}>
                {t('tx_history')}
              </h3>
              {walletQuery.isLoading ? (
                <LoadingSpinner />
              ) : txs.length === 0 ? (
                <p className="muted" style={{ padding: '16px 4px' }}>
                  {t('no_tx')}
                </p>
              ) : (
                txs.map((tx) => (
                  <div className="lrow" key={tx.id}>
                    <span
                      className="icon-btn"
                      style={{
                        width: 38,
                        height: 38,
                        color: tx.amount > 0 ? 'var(--ok)' : 'var(--text-dim)',
                      }}
                    >
                      <Icon name={tx.amount > 0 ? 'plus' : 'minus'} size={16} />
                    </span>
                    <div className="col" style={{ gap: 1, flex: 1 }}>
                      <span style={{ fontWeight: 700, fontSize: 14 }}>{txLabel(tx, t)}</span>
                      <span className="tiny faint num">{fmtTxDate(tx.createdAt)}</span>
                    </div>
                    <span
                      className="num"
                      style={{ fontWeight: 800, color: tx.amount > 0 ? 'var(--ok)' : 'var(--text)' }}
                    >
                      {tx.amount > 0 ? '+' : '−'}
                      {fmtPrice(Math.abs(tx.amount), cur)}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
