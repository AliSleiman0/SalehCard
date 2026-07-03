import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Icon,
  PageHead,
  Art,
  StatusBadge,
  FfBadge,
  PayChip,
  Chip,
  Tabs,
  Segmented,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money, relativeTime } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'
import { useProducts } from '@/features/products/hooks/useProducts'
import { updateProduct } from '@/features/products/api/products'
import type { Product, Variant } from '@/types'
import {
  useReseller,
  useTiers,
  useUpdateResellerTier,
  useAdjustResellerBalance,
} from '../hooks/useResellers'
import { adaptReseller, tierColor } from '../lib/adaptReseller'

type Tab = 'overview' | 'balance' | 'pricing' | 'orders'

export default function ResellerDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { data, isLoading, isError, refetch } = useReseller(id)
  const [tab, setTab] = useState<Tab>('overview')

  const detail = data?.data
  const view = detail ? adaptReseller(detail) : null

  if (isLoading) return <LoadingSpinner />
  if (isError || !detail || !view) {
    return (
      <div className="page">
        <ErrorState message="Couldn't load this reseller." onRetry={() => refetch()} />
      </div>
    )
  }

  const tc = tierColor(view.tier)

  return (
    <div className="page">
      <PageHead
        crumbs={[t('nav_resellers'), view.name]}
        title={
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 12 }}>
            {view.name}{' '}
            {view.tier ? (
              <span className="pill-role" style={{ background: tc + '22', color: tc, border: '1px solid ' + tc + '55' }}>
                {view.tier} agent
              </span>
            ) : (
              <span className="faint" style={{ fontSize: 13 }}>
                no tier
              </span>
            )}
          </span>
        }
        sub={view.email || view.phone}
      >
        <button className="abtn" onClick={() => navigate('/resellers')}>
          <Icon name="chevleft" size={15} /> {t('back')}
        </button>
        <button className="abtn primary" onClick={() => setTab('balance')}>
          <Icon name="coins" size={15} /> Adjust sub-balance
        </button>
      </PageHead>

      <div className="g3" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 18 }}>
        {(
          [
            ['Sub-balance', money(view.balance, view.cur), 'wallet'],
            ['Tier margin', view.margin + '%', 'activity'],
            ['Total orders', view.orders.toLocaleString(), 'bag'],
            ['Lifetime volume', money(view.vol, view.cur), 'coins'],
          ] as [string, string, 'wallet' | 'activity' | 'bag' | 'coins'][]
        ).map(([l, v, ic]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: tc + '22', color: tc }}>
                <Icon name={ic} size={16} />
              </div>
              <div className="k-label">{l}</div>
            </div>
            <div className="k-val" style={{ fontSize: 22 }}>
              {v}
            </div>
          </div>
        ))}
      </div>

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'overview', label: 'Overview' },
          { k: 'balance', label: 'Sub-balance' },
          { k: 'pricing', label: 'Pricing rules' },
          { k: 'orders', label: 'Order history' },
        ]}
      />

      {tab === 'overview' && (
        <div className="formgrid">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Profile</h3>
            <div className="deflist">
              {(
                [
                  ['Display name', view.name],
                  ['Email', view.email || '—'],
                  ['Phone', view.phone || '—'],
                  ['Reseller ID', view.id],
                  ['Status', <StatusBadge key="s" s={view.status} />],
                  ['Currency', view.cur],
                  ['Member since', view.joined],
                ] as [string, React.ReactNode][]
              ).map(([k, v]) => (
                <div className="defrow" key={k}>
                  <span className="dk">{k}</span>
                  <span className="dv">{v}</span>
                </div>
              ))}
            </div>
          </div>
          <TierTab id={view.id} tier={view.tier} margin={view.margin} />
        </div>
      )}

      {tab === 'balance' && <BalanceTab id={view.id} balance={view.balance} transactions={detail.transactions} />}

      {tab === 'pricing' && <PricingTab margin={view.margin} />}

      {tab === 'orders' && <OrdersTab email={view.email} total={view.orders} />}
    </div>
  )
}

/** Tier assignment — pick a tier and persist via the admin endpoint. */
function TierTab({ id, tier, margin }: { id: string; tier: string; margin: number }) {
  const { t } = useTranslation()
  const [pending, setPending] = useState(tier)
  const tiersQuery = useTiers()
  const tiers = tiersQuery.data?.data ?? []
  const update = useUpdateResellerTier(id)

  // Keep the selection in sync if the underlying tier changes (refetch).
  useEffect(() => setPending(tier), [tier])

  return (
    <div className="acard pad">
      <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>Tier assignment</h3>
      <div className="g3">
        {tiers.map((tn) => (
          <Chip
            key={tn.id}
            on={pending === tn.name}
            onClick={() => setPending(tn.name)}
            style={{ justifyContent: 'center', padding: 12 }}
          >
            {tn.name}
          </Chip>
        ))}
      </div>
      <div className="ahint">Tier sets the default discount ({margin}%) and balance limit.</div>
      <div style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center' }}>
        <button
          className="abtn primary"
          disabled={pending === tier || update.isPending}
          onClick={() => update.mutate(pending)}
        >
          <Icon name="check" size={15} /> {t('save')}
        </button>
        {update.isError && <span style={{ color: 'var(--danger)', fontSize: 12.5 }}>Couldn't update tier.</span>}
      </div>
    </div>
  )
}

/** Sub-balance ledger + adjust form. Sub-balance is the reseller's wallet. */
function BalanceTab({
  id,
  balance,
  transactions,
}: {
  id: string
  balance: number
  transactions: { id: string; type: string; amount: number; method: string; createdAt: string }[]
}) {
  const [dir, setDir] = useState<'topup' | 'deduct'>('topup')
  const [amount, setAmount] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const adjust = useAdjustResellerBalance(id)

  const apply = () => {
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) {
      setError('Enter an amount greater than zero.')
      return
    }
    if (!reason.trim()) {
      setError('A note is required.')
      return
    }
    setError('')
    adjust.mutate(
      { direction: dir === 'topup' ? 'credit' : 'debit', amount: amt, reason: reason.trim() },
      {
        onSuccess: () => {
          setAmount('')
          setReason('')
        },
        onError: (e) => setError(e instanceof ApiError ? e.message : 'Adjustment failed.'),
      },
    )
  }

  return (
    <div className="formgrid">
      <div className="acard">
        <div className="panelhead">
          <Icon name="wallet" size={17} />
          <h3>Sub-balance transactions</h3>
        </div>
        {transactions.length === 0 ? (
          <EmptyState title="No sub-balance activity yet" />
        ) : (
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Method</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx) => (
                  <tr key={tx.id}>
                    <td className="mono strong">{tx.id.slice(-8)}</td>
                    <td className="muted" style={{ textTransform: 'capitalize' }}>
                      {tx.type}
                    </td>
                    <td className="num strong" style={{ color: tx.amount < 0 ? 'var(--danger)' : 'var(--ok)' }}>
                      {tx.amount < 0 ? '−' : '+'}
                      {money(Math.abs(tx.amount)).replace('−', '')}
                    </td>
                    <td>{tx.method ? <PayChip p={tx.method} /> : <span className="faint">—</span>}</td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {relativeTime(tx.createdAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
      <div className="acard pad">
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 4 }}>Adjust sub-balance</h3>
        <div className="faint" style={{ fontSize: 12.5, marginBottom: 14 }}>
          Current {money(balance)}
        </div>
        <Segmented<'topup' | 'deduct'>
          block
          value={dir}
          onChange={setDir}
          items={[
            { k: 'topup', label: 'Top up (+)' },
            { k: 'deduct', label: 'Deduct (−)' },
          ]}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          Amount (USD)
        </label>
        <input
          className="afield"
          type="number"
          min="0"
          step="0.01"
          placeholder="0.00"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <label className="alabel" style={{ marginTop: 12 }}>
          Note (required)
        </label>
        <input
          className="afield"
          placeholder="Reference / reason"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <button className="abtn primary" style={{ marginTop: 16, width: '100%' }} onClick={apply} disabled={adjust.isPending}>
          <Icon name="check" size={15} /> Apply
        </button>
      </div>
    </div>
  )
}

/** Effective reseller price = the lowest of retail, the tier-margin price, and
 *  any explicit per-variant override (mirrors the backend order pricing). */
function effectivePrice(v: Variant, margin: number): number {
  let p = v.price
  if (margin > 0 && margin < 100) p = Math.min(p, v.price * (1 - margin / 100))
  if (v.resellerPrice != null) p = Math.min(p, v.resellerPrice)
  return p
}

/** Pricing rules — a read-mostly effective-price table for this reseller's tier,
 *  with inline editing of each variant's per-product reseller-price override
 *  (persisted via the product-update endpoint; no new backend). */
function PricingTab({ margin }: { margin: number }) {
  const qc = useQueryClient()
  const { data, isLoading, isError, refetch } = useProducts({ limit: 100 })
  const products = data?.data ?? []
  const [edit, setEdit] = useState<{ variantId: string } | null>(null)
  const [value, setValue] = useState('')
  const [error, setError] = useState('')

  const save = useMutation({
    mutationFn: ({ product, variantId, price }: { product: Product; variantId: string; price?: number }) =>
      updateProduct(product.id, {
        variants: product.variants.map((v) => ({
          denomination: v.denomination,
          price: v.price,
          resellerPrice: v.id === variantId ? price : v.resellerPrice,
        })),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'products'] })
      setEdit(null)
      setError('')
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : 'Update failed.'),
  })

  const startEdit = (v: Variant) => {
    setEdit({ variantId: v.id })
    setValue(v.resellerPrice != null ? String(v.resellerPrice) : '')
    setError('')
  }
  const commit = (product: Product, variantId: string) => {
    const raw = value.trim()
    const price = raw === '' ? undefined : Number(raw)
    if (price !== undefined && (!Number.isFinite(price) || price < 0)) {
      setError('Enter a valid price, or clear it to remove the override.')
      return
    }
    save.mutate({ product, variantId, price })
  }

  if (isLoading) return <LoadingSpinner />
  if (isError) return <ErrorState message="Couldn't load products." onRetry={() => refetch()} />

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="coins" size={17} />
        <h3>Effective pricing ({margin}% tier margin)</h3>
      </div>
      <div className="ahint" style={{ padding: '0 16px 12px' }}>
        Resellers pay the lowest of retail, the tier-margin price, and any per-variant override below.
      </div>
      {products.length === 0 ? (
        <EmptyState title="No products yet" />
      ) : (
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Product</th>
                <th>Variant</th>
                <th>Retail</th>
                <th>Tier margin</th>
                <th>Override</th>
                <th>Effective</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {products.flatMap((p) =>
                p.variants.map((v) => {
                  const marginPrice = margin > 0 && margin < 100 ? v.price * (1 - margin / 100) : v.price
                  const editing = edit?.variantId === v.id
                  return (
                    <tr key={v.id}>
                      <td>
                        <b style={{ fontSize: 12.5 }}>{p.title.en}</b>
                      </td>
                      <td className="muted">{v.denomination}</td>
                      <td className="num">{money(v.price)}</td>
                      <td className="num muted">{money(marginPrice)}</td>
                      <td className="num">
                        {editing ? (
                          <input
                            className="afield"
                            type="number"
                            min="0"
                            step="0.01"
                            placeholder="none"
                            value={value}
                            autoFocus
                            onChange={(e) => setValue(e.target.value)}
                            style={{ width: 90, padding: '4px 8px' }}
                          />
                        ) : v.resellerPrice != null ? (
                          money(v.resellerPrice)
                        ) : (
                          <span className="faint">—</span>
                        )}
                      </td>
                      <td className="num strong">{money(effectivePrice(v, margin))}</td>
                      <td onClick={(e) => e.stopPropagation()}>
                        {editing ? (
                          <div className="row-actions">
                            <button className="abtn xs primary" disabled={save.isPending} onClick={() => commit(p, v.id)}>
                              <Icon name="check" size={13} />
                            </button>
                            <button className="abtn xs" disabled={save.isPending} onClick={() => setEdit(null)}>
                              <Icon name="x" size={13} />
                            </button>
                          </div>
                        ) : (
                          <span className="iact" onClick={() => startEdit(v)}>
                            <Icon name="edit" size={15} />
                          </span>
                        )}
                      </td>
                    </tr>
                  )
                }),
              )}
            </tbody>
          </table>
        </div>
      )}
      {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, padding: '10px 16px' }}>{error}</div>}
    </div>
  )
}

/** Order history — reuses the wired admin orders endpoint, searched by email. */
function OrdersTab({ email, total }: { email: string; total: number }) {
  const navigate = useNavigate()
  const { data, isLoading, isError } = useOrders({ q: email, limit: 6 })
  const rows = (data?.data ?? []).map(adaptOrder)

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="bag" size={17} />
        <h3>Order history</h3>
        <span className="faint" style={{ fontSize: 12.5, marginInlineStart: 6 }}>
          {total} completed
        </span>
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Couldn't load orders." />
      ) : rows.length === 0 ? (
        <EmptyState title="No orders yet" />
      ) : (
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Order</th>
                <th>Product</th>
                <th>Type</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Date</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((o) => (
                <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                  <td className="mono strong">{o.id.slice(-8)}</td>
                  <td>
                    <div className="cellprod">
                      <Art art={o.art} size={28} radius={6} />
                      <div className="pn">
                        <b style={{ fontSize: 12.5 }}>{o.product}</b>
                      </div>
                    </div>
                  </td>
                  <td>
                    <FfBadge ff={o.ff} />
                  </td>
                  <td className="num strong">{money(o.amount, o.cur)}</td>
                  <td>
                    <StatusBadge s={o.status} />
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {o.date}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
