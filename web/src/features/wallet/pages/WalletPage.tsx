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
import { useTopUp, useTopUpMethods, useTopUpRequests } from '../hooks/useTopUp'
import { MethodFields } from '../components/MethodFields'
import { methodFieldPayload, missingRequired, type MethodValues } from '../lib/methodFields'
import type { TopUpMethod } from '../api/wallet'
import type { WalletTransaction } from '@/types'

const PRESETS = [25, 50, 100, 250]

// Selection key for the on-chain USDT option (settles automatically, distinct
// from the admin-defined manual methods — mirrors the app's top-up screen).
const USDT_KEY = '__usdt_onchain__'

// Badge colour for a method chip: known brands keep their colour, anything else
// gets a stable colour from its name.
const BRAND_COLORS: [RegExp, string][] = [
  [/whish/i, '#e0195c'],
  [/omt/i, '#f39200'],
  [/sham/i, '#1e6fd9'],
  [/usdt|tether|crypto/i, '#26a17b'],
  [/bank|iban/i, '#3b4cca'],
]
const PALETTE = ['#7c3aed', '#0ea5e9', '#14b8a6', '#f43f5e', '#f59e0b', '#64748b']
function methodColor(name: string): string {
  for (const [re, c] of BRAND_COLORS) if (re.test(name)) return c
  let h = 0
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return PALETTE[h % PALETTE.length]
}

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
  const methodsQuery = useTopUpMethods()
  const keyRef = useRef<string | null>(null)
  const [amt, setAmt] = useState(50)
  const [amtRaw, setAmtRaw] = useState('')
  const [sel, setSel] = useState<string | null>(null)
  const [values, setValues] = useState<MethodValues>({})
  const [uploading, setUploading] = useState(false)
  const [usdtNetwork, setUsdtNetwork] = useState('')

  const balance = walletQuery.data?.balance ?? 0
  const txs = walletQuery.data?.transactions ?? []
  const requests = requestsQuery.data ?? []
  const usdtEnabled = paymentConfig.data?.usdtEnabled === true
  const networks = paymentConfig.data?.networks ?? []
  const methods: TopUpMethod[] = methodsQuery.data ?? []
  // Options: on-chain USDT first when enabled, then the admin's manual methods.
  const options: { key: string; name: string; method?: TopUpMethod }[] = [
    ...(usdtEnabled ? [{ key: USDT_KEY, name: 'USDT' }] : []),
    ...methods.map((m) => ({ key: m.id, name: m.name, method: m })),
  ]
  // Default to the first option until the customer picks one (or when the admin
  // removed the picked method).
  const selKey = options.some((o) => o.key === sel) ? sel : (options[0]?.key ?? null)
  const selected = options.find((o) => o.key === selKey)
  // USDT chosen (only offered when on-chain is enabled) → the instant deposit flow.
  const usdtInstant = selKey === USDT_KEY
  const method = selected?.method

  const pick = (key: string) => {
    if (key === selKey) return
    setSel(key)
    setValues({}) // inputs belong to the previously-selected method
  }

  const onTopUp = (): void => {
    if (topUp.isPending || createIntent.isPending || uploading) return
    if (!(amt > 0)) {
      toast(t('amount'), 'wallet')
      return
    }

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

    if (!method) {
      toast('Please choose a payment method.', 'wallet')
      return
    }
    const missing = missingRequired(method.fields, values)
    if (missing.length > 0) {
      toast(`${t('field_required')}: ${missing.join(', ')}`, 'user')
      return
    }
    const fields = methodFieldPayload(method.fields, values)
    topUp.mutate(
      { amount: amt, methodId: method.id, fields },
      {
        onSuccess: () => {
          setValues({})
          toast('Request submitted — your wallet is credited once we confirm your payment.', 'wallet')
        },
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
                      variant={amt === p && !amtRaw ? 'primary' : 'ghost'}
                      size="sm"
                      onClick={() => {
                        setAmt(p)
                        setAmtRaw('')
                      }}
                    >
                      {fmtPrice(p, cur)}
                    </Button>
                  ))}
                </div>
                <input
                  className="field num"
                  inputMode="decimal"
                  aria-label={t('amount')}
                  placeholder={t('amount')}
                  value={amtRaw}
                  onChange={(e) => {
                    const raw = e.target.value.replace(/[^0-9.]/g, '')
                    setAmtRaw(raw)
                    const n = parseFloat(raw)
                    setAmt(Number.isFinite(n) ? n : 0)
                  }}
                  style={{ marginBottom: 14 }}
                />
                <div className="label">{t('topup_via')}</div>
                <div className="row wrap-gap" style={{ gap: 10, marginBottom: 12 }}>
                  {options.map((o) => (
                    <div
                      key={o.key}
                      role="button"
                      className={'method' + (selKey === o.key ? ' on' : '')}
                      style={{ flex: 1, padding: 12, minWidth: 110 }}
                      onClick={() => pick(o.key)}
                    >
                      <span className="mi" style={{ background: methodColor(o.name) }}>
                        {o.key === USDT_KEY ? '₮' : o.name.trim().charAt(0).toUpperCase()}
                      </span>
                      <span style={{ fontWeight: 700 }}>{o.name}</span>
                    </div>
                  ))}
                  {methodsQuery.isLoading && <LoadingSpinner />}
                  {!methodsQuery.isLoading && options.length === 0 && (
                    <p className="small muted">No top-up methods are available right now — please contact support.</p>
                  )}
                </div>
                {method && method.instructions.trim() && (
                  <div className="panel card-pad" style={{ padding: 12, marginBottom: 12, whiteSpace: 'pre-line' }}>
                    <span className="small">{method.instructions}</span>
                  </div>
                )}
                {method && method.fields.length > 0 && (
                  <MethodFields
                    key={method.id}
                    fields={method.fields}
                    values={values}
                    onChange={(k, v) => setValues((cur) => ({ ...cur, [k]: v }))}
                    onUploadingChange={setUploading}
                  />
                )}
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
                  disabled={topUp.isPending || createIntent.isPending || uploading || !selected}
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
                            <span className="tiny faint">· {r.methodName || r.channel.toUpperCase()}</span>
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
