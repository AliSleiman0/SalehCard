import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, Button, useToast, LoadingSpinner } from '@/components'
import { AcctSidebar } from '@/features/auth/components/AcctSidebar'
import { useCurrencyStore } from '@/stores/currency'
import { fmtPrice } from '@/lib/utils'
import { useWallet } from '../hooks/useWallet'
import { useTopUp } from '../hooks/useTopUp'
import type { WalletTransaction } from '@/types'

const PRESETS = [25, 50, 100, 250]

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
  const cur = useCurrencyStore((s) => s.currency)
  const walletQuery = useWallet()
  const topUp = useTopUp()
  const [amt, setAmt] = useState(50)
  const [via, setVia] = useState<'visa' | 'usdt'>('visa')

  const balance = walletQuery.data?.balance ?? 0
  const txs = walletQuery.data?.transactions ?? []

  const onTopUp = (): void => {
    if (topUp.isPending) return
    topUp.mutate(
      { amount: amt, method: via === 'visa' ? 'card' : 'usdt' },
      {
        onSuccess: () => toast(`+${fmtPrice(amt, cur)}`, 'wallet'),
        onError: (err) => toast(err.message || 'Top-up failed', 'user'),
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
                <span className="badge" style={{ background: 'rgba(255,255,255,.16)', color: '#fff' }}>
                  <Icon name="bolt" size={12} />
                  {t('trust_instant')}
                </span>
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
                <div className="row" style={{ gap: 10, marginBottom: 16 }}>
                  <div
                    className={'method' + (via === 'visa' ? ' on' : '')}
                    style={{ flex: 1, padding: 12 }}
                    onClick={() => setVia('visa')}
                  >
                    <span className="mi" style={{ background: '#1a1f71' }}>
                      VISA
                    </span>
                    <span style={{ fontWeight: 700 }}>{t('pay_visa')}</span>
                  </div>
                  <div
                    className={'method' + (via === 'usdt' ? ' on' : '')}
                    style={{ flex: 1, padding: 12 }}
                    onClick={() => setVia('usdt')}
                  >
                    <span className="mi" style={{ background: '#26a17b' }}>
                      ₮
                    </span>
                    <span style={{ fontWeight: 700 }}>USDT</span>
                  </div>
                </div>
                <Button
                  variant="primary"
                  size="lg"
                  block
                  disabled={topUp.isPending}
                  onClick={onTopUp}
                >
                  {topUp.isPending ? t('processing') : `${t('topup')} ${fmtPrice(amt, cur)}`}
                </Button>
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
