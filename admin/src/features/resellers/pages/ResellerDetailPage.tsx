import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
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
import type { Variant } from '@/types'
import {
  useReseller,
  useTiers,
  useUpdateResellerTier,
  useAdjustResellerBalance,
  useResellerPrices,
  useSetResellerPrice,
  useDeleteResellerPrice,
} from '../hooks/useResellers'
import { adaptReseller, effectivePrice, tierColor } from '../lib/adaptReseller'

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
        <ErrorState message={t('rd_load_error')} onRetry={() => refetch()} />
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
                {view.tier} {t('rd_agent')}
              </span>
            ) : (
              <span className="faint" style={{ fontSize: 13 }}>
                {t('rd_no_tier')}
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
          <Icon name="coins" size={15} /> {t('rd_adjust_balance')}
        </button>
      </PageHead>

      <div className="g3" style={{ gridTemplateColumns: 'repeat(4,1fr)', marginBottom: 18 }}>
        {(
          [
            [t('rd_kpi_subbalance'), money(view.balance, view.cur), 'wallet'],
            [t('col_tier_margin'), view.margin + '%', 'activity'],
            [t('rd_kpi_total_orders'), view.orders.toLocaleString(), 'bag'],
            [t('rd_kpi_lifetime_volume'), money(view.vol, view.cur), 'coins'],
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
          { k: 'overview', label: t('grp_overview') },
          { k: 'balance', label: t('rd_kpi_subbalance') },
          { k: 'pricing', label: t('rd_tab_pricing') },
          { k: 'orders', label: t('rd_tab_orders') },
        ]}
      />

      {tab === 'overview' && (
        <div className="formgrid">
          <div className="acard pad">
            <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('rd_profile')}</h3>
            <div className="deflist">
              {(
                [
                  [t('rd_display_name'), view.name],
                  [t('login_email'), view.email || '—'],
                  [t('rd_phone'), view.phone || '—'],
                  [t('rd_reseller_id'), view.id],
                  [t('status'), <StatusBadge key="s" s={view.status} />],
                  [t('exp_currency'), view.cur],
                  [t('rd_member_since'), view.joined],
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

      {tab === 'pricing' && <PricingTab id={view.id} margin={view.margin} />}

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
      <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 14 }}>{t('rd_tier_assignment')}</h3>
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
      <div className="ahint">{t('rd_tier_hint', { margin })}</div>
      <div style={{ marginTop: 16, display: 'flex', gap: 10, alignItems: 'center' }}>
        <button
          className="abtn primary"
          disabled={pending === tier || update.isPending}
          onClick={() => update.mutate(pending)}
        >
          <Icon name="check" size={15} /> {t('save')}
        </button>
        {update.isError && <span style={{ color: 'var(--danger)', fontSize: 12.5 }}>{t('rd_tier_update_error')}</span>}
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
  const { t } = useTranslation()
  const [dir, setDir] = useState<'topup' | 'deduct'>('topup')
  const [amount, setAmount] = useState('')
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const adjust = useAdjustResellerBalance(id)

  const apply = () => {
    const amt = Number(amount)
    if (!Number.isFinite(amt) || amt <= 0) {
      setError(t('rd_err_amount_positive'))
      return
    }
    if (!reason.trim()) {
      setError(t('rd_err_note_required'))
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
        onError: (e) => setError(e instanceof ApiError ? e.message : t('rd_adjust_failed')),
      },
    )
  }

  return (
    <div className="formgrid">
      <div className="acard">
        <div className="panelhead">
          <Icon name="wallet" size={17} />
          <h3>{t('rd_subbalance_transactions')}</h3>
        </div>
        {transactions.length === 0 ? (
          <EmptyState title={t('rd_no_subbalance_activity')} />
        ) : (
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('col_transaction')}</th>
                  <th>{t('col_type')}</th>
                  <th>{t('col_amount')}</th>
                  <th>{t('col_method')}</th>
                  <th>{t('col_date')}</th>
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
        <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 4 }}>{t('rd_adjust_balance')}</h3>
        <div className="faint" style={{ fontSize: 12.5, marginBottom: 14 }}>
          {t('rd_current', { amount: money(balance) })}
        </div>
        <Segmented<'topup' | 'deduct'>
          block
          value={dir}
          onChange={setDir}
          items={[
            { k: 'topup', label: t('rd_topup_plus') },
            { k: 'deduct', label: t('rd_deduct_minus') },
          ]}
        />
        <label className="alabel" style={{ marginTop: 14 }}>
          {t('rd_amount_usd')}
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
          {t('rd_note_required_label')}
        </label>
        <input
          className="afield"
          placeholder={t('rd_ph_reason')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <button className="abtn primary" style={{ marginTop: 16, width: '100%' }} onClick={apply} disabled={adjust.isPending}>
          <Icon name="check" size={15} /> {t('apply')}
        </button>
      </div>
    </div>
  )
}

/** Pricing rules — the effective-price table for this reseller. The inline
 *  editor sets THIS reseller's per-variant custom price (the /prices endpoints,
 *  the most specific pricing layer); the global per-variant override is shown
 *  read-only and lives in the product editor. */
function PricingTab({ id, margin }: { id: string; margin: number }) {
  const { t } = useTranslation()
  const { data, isLoading, isError, refetch } = useProducts({ limit: 100 })
  const prices = useResellerPrices(id)
  const products = data?.data ?? []
  const custom = new Map((prices.data?.data ?? []).map((p) => [p.variantId, p.price]))
  const [edit, setEdit] = useState<{ variantId: string } | null>(null)
  const [value, setValue] = useState('')
  const [error, setError] = useState('')
  const setPrice = useSetResellerPrice(id)
  const clearPrice = useDeleteResellerPrice(id)

  const done = () => {
    setEdit(null)
    setError('')
  }
  const fail = (e: unknown) => setError(e instanceof ApiError ? e.message : t('rd_update_failed'))

  const startEdit = (v: Variant) => {
    setEdit({ variantId: v.id })
    setValue(custom.has(v.id) ? String(custom.get(v.id)) : '')
    setError('')
  }
  const commit = (productId: string, variantId: string) => {
    const raw = value.trim()
    if (raw === '') {
      // Clearing: remove this reseller's custom row (idempotent server-side).
      if (!custom.has(variantId)) {
        done()
        return
      }
      clearPrice.mutate(variantId, { onSuccess: done, onError: fail })
      return
    }
    const price = Number(raw)
    if (!Number.isFinite(price) || price < 0) {
      setError(t('rd_err_valid_price'))
      return
    }
    setPrice.mutate({ productId, variantId, price }, { onSuccess: done, onError: fail })
  }
  const saving = setPrice.isPending || clearPrice.isPending

  if (isLoading || prices.isLoading) return <LoadingSpinner />
  if (isError) return <ErrorState message={t('rd_load_products_error')} onRetry={() => refetch()} />
  if (prices.isError) return <ErrorState message={t('rd_load_products_error')} onRetry={() => prices.refetch()} />

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="coins" size={17} />
        <h3>{t('rd_effective_pricing', { margin })}</h3>
      </div>
      <div className="ahint" style={{ padding: '0 16px 12px' }}>
        {t('rd_pricing_hint')}
      </div>
      {products.length === 0 ? (
        <EmptyState title={t('rd_no_products')} />
      ) : (
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>{t('col_product')}</th>
                <th>{t('col_variant')}</th>
                <th>{t('col_retail')}</th>
                <th>{t('col_tier_margin')}</th>
                <th>{t('col_override')}</th>
                <th>{t('col_custom')}</th>
                <th>{t('col_effective')}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {products.flatMap((p) =>
                p.variants.map((v) => {
                  const marginPrice = margin > 0 && margin < 100 ? v.price * (1 - margin / 100) : v.price
                  const editing = edit?.variantId === v.id
                  const customPrice = custom.get(v.id)
                  return (
                    <tr key={v.id}>
                      <td>
                        <b style={{ fontSize: 12.5 }}>{p.title.en}</b>
                      </td>
                      <td className="muted">{v.denomination}</td>
                      <td className="num">{money(v.price)}</td>
                      <td className="num muted">{money(marginPrice)}</td>
                      <td className="num muted">
                        {v.resellerPrice != null ? money(v.resellerPrice) : <span className="faint">—</span>}
                      </td>
                      <td className="num">
                        {editing ? (
                          <input
                            className="afield"
                            type="number"
                            min="0"
                            step="0.01"
                            placeholder={t('rd_ph_none')}
                            value={value}
                            autoFocus
                            onChange={(e) => setValue(e.target.value)}
                            style={{ width: 90, padding: '4px 8px' }}
                          />
                        ) : customPrice != null ? (
                          money(customPrice)
                        ) : (
                          <span className="faint">—</span>
                        )}
                      </td>
                      <td className="num strong">{money(effectivePrice(v, margin, customPrice))}</td>
                      <td onClick={(e) => e.stopPropagation()}>
                        {editing ? (
                          <div className="row-actions">
                            <button className="abtn xs primary" disabled={saving} onClick={() => commit(p.id, v.id)}>
                              <Icon name="check" size={13} />
                            </button>
                            <button className="abtn xs" disabled={saving} onClick={() => setEdit(null)}>
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
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { data, isLoading, isError } = useOrders({ q: email, limit: 6 })
  const rows = (data?.data ?? []).map(adaptOrder)

  return (
    <div className="acard">
      <div className="panelhead">
        <Icon name="bag" size={17} />
        <h3>{t('rd_tab_orders')}</h3>
        <span className="faint" style={{ fontSize: 12.5, marginInlineStart: 6 }}>
          {t('rd_completed_count', { total })}
        </span>
      </div>
      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('rd_load_orders_error')} />
      ) : rows.length === 0 ? (
        <EmptyState title={t('rd_no_orders')} />
      ) : (
        <div className="tablewrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>{t('col_order')}</th>
                <th>{t('col_product')}</th>
                <th>{t('col_type')}</th>
                <th>{t('col_amount')}</th>
                <th>{t('status')}</th>
                <th>{t('col_date')}</th>
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
