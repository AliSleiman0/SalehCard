import { useEffect, useMemo, useState } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Avatar,
  PayChip,
  Modal,
  Chip,
  RoleBadge,
  AreaChart,
  Donut,
  Bars,
  Pagination,
  Tabs,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money, downloadCsv, downloadPdf, formatDateTime } from '@/lib/utils'
import { apiErrorText } from '@/lib/apiErrorText'
import { useCan } from '@/stores/auth'
import { useTransactions, useRevenueSummary } from '../hooks/useFinance'
import { adaptTx } from '../lib/adaptFinance'
import { listTransactions, type TxType, type LabelValue, type RevenueSummary } from '../api/finance'
import { useTopUps, useApproveTopUp, useRejectTopUp } from '@/features/topups/hooks/useTopups'
import type { AdminTopUp, TopUpStatus } from '@/features/topups/api/topups'

type Tab = 'transactions' | 'revenue' | 'usdt'
type Range = 'daily' | 'weekly' | 'monthly'

const TX_COLOR: Record<string, string> = {
  topup: 'var(--ok)',
  purchase: 'var(--ff-code)',
  refund: 'var(--warn)',
  adjustment: 'var(--brand-2)',
}

// Display metadata for each payment method the revenue donut breaks down by. The
// API groups by the stored method (card/wallet/usdt); card renders as Visa.
const METHOD_META: Record<string, { labelKey: string; color: string }> = {
  wallet: { labelKey: 'fin_method_store_wallet', color: '#8a3bff' },
  card: { labelKey: 'fin_method_visa', color: '#3b5bff' },
  usdt: { labelKey: 'fin_method_usdt', color: '#22e3c8' },
}

export default function FinancePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('transactions')
  const [exporting, setExporting] = useState(false)
  // The USDT queue reuses the top-up endpoints (topups domain) — hide the tab
  // when the admin's role lacks topups.view.
  const canViewTopups = useCan()('topups.view')

  // Export the full wallet-ledger feed (all pages — the backend caps limit at
  // 100, so page to total). CSV mirrors the transactions table.
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const all = []
      let p = 1
      let pages = 1
      do {
        const res = await listTransactions({ page: p, limit: 100 })
        all.push(...(res.data ?? []))
        pages = res.meta?.pages ?? 1
        p++
      } while (p <= pages)

      const header = [
        t('fin_csv_tx_id'),
        t('fin_user'),
        t('fin_csv_role'),
        t('col_type'),
        t('col_amount'),
        t('fin_csv_balance_after'),
        t('col_method'),
        t('fin_csv_ref'),
        t('col_date'),
      ]
      const csvRows = all.map((tx) => [
        tx.id,
        tx.user?.name ?? '',
        tx.user?.role ?? '',
        tx.type,
        tx.amount.toFixed(2),
        tx.balanceAfter.toFixed(2),
        tx.method,
        tx.ref,
        new Date(tx.createdAt).toISOString(),
      ])
      downloadCsv(`transactions-${new Date().toISOString().slice(0, 10)}.csv`, [header, ...csvRows])
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="page page-wide">
      <PageHead
        back="/"
        crumbs={[t('grp_finance'), t('nav_wallet')]}
        title={t('fin_title')}
        sub={t('fin_sub')}
      >
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
      </PageHead>

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'transactions', label: t('fin_tab_transactions') },
          { k: 'revenue', label: t('fin_tab_revenue') },
          ...(canViewTopups ? [{ k: 'usdt' as const, label: t('fin_tab_usdt') }] : []),
        ]}
      />

      {tab === 'transactions' && <TransactionsTab />}
      {tab === 'revenue' && <RevenueTab />}
      {tab === 'usdt' && canViewTopups && <UsdtQueueTab />}
    </div>
  )
}

/** The wallet-ledger feed: filter by type/method, search by user, paginate. */
function TransactionsTab() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const [type, setType] = useState<'' | TxType>('')
  const [method, setMethod] = useState('')
  const [search, setSearch] = useState(searchParams.get('q') ?? '')
  const [debouncedSearch, setDebouncedSearch] = useState(search)
  const [page, setPage] = useState(1)

  // Adopt the URL's ?q= (e.g. the top-bar global search navigates to /finance?q=…).
  useEffect(() => {
    setSearch(searchParams.get('q') ?? '')
  }, [searchParams])

  // Debounce the search term so we issue one request per pause, not per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, isError, refetch } = useTransactions({
    page,
    type: type || undefined,
    method: method || undefined,
    q: debouncedSearch.trim(),
  })

  const rows = useMemo(() => (data?.data ?? []).map(adaptTx), [data])
  const meta = data?.meta

  return (
    <div className="acard">
      <div className="toolbar">
        <div className="fsearch">
          <Icon name="search" size={15} />
          <input
            placeholder={t('fin_search_ph')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <select
          className="select"
          value={type}
          onChange={(e) => {
            setType(e.target.value as '' | TxType)
            setPage(1)
          }}
        >
          <option value="">{t('fin_all_types')}</option>
          <option value="topup">{t('fin_type_topup')}</option>
          <option value="purchase">{t('fin_type_purchase')}</option>
          <option value="refund">{t('fin_type_refund')}</option>
          <option value="adjustment">{t('fin_type_adjustment')}</option>
        </select>
        <select
          className="select"
          value={method}
          onChange={(e) => {
            setMethod(e.target.value)
            setPage(1)
          }}
        >
          <option value="">{t('fin_all_methods')}</option>
          <option value="wallet">{t('fin_method_wallet')}</option>
          <option value="card">{t('fin_method_card')}</option>
          <option value="usdt">{t('fin_method_usdt')}</option>
        </select>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message={t('fin_load_error')} onRetry={() => refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState title={t('fin_empty')} />
      ) : (
        <>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('col_transaction')}</th>
                  <th>{t('fin_user')}</th>
                  <th>{t('col_type')}</th>
                  <th>{t('col_amount')}</th>
                  <th>{t('col_method')}</th>
                  <th>{t('col_date')}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((tx) => (
                  <tr key={tx.id}>
                    <td className="mono strong">{tx.id.slice(-8)}</td>
                    <td>
                      <div className="cellprod">
                        <Avatar name={tx.raw.user?.name || t('fin_unknown_user')} />
                        <div className="pn">
                          <b style={{ fontSize: 12.5 }}>{tx.raw.user?.name || t('fin_unknown_user')}</b>
                          {tx.role && (
                            <span className={'pill-role role-' + tx.role} style={{ fontSize: 10 }}>
                              {t(`fin_role_${tx.role}`, { defaultValue: tx.role })}
                            </span>
                          )}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span
                        style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 6,
                          fontWeight: 700,
                          fontSize: 12.5,
                          textTransform: 'capitalize',
                        }}
                      >
                        <span style={{ width: 7, height: 7, borderRadius: 99, background: TX_COLOR[tx.type] }} />
                        {t(`fin_tx_${tx.type}`, { defaultValue: tx.type })}
                      </span>
                    </td>
                    <td className="num strong" style={{ color: tx.amount < 0 ? 'var(--danger)' : 'var(--ok)' }}>
                      {tx.amount < 0 ? '−' : '+'}
                      {money(Math.abs(tx.amount)).replace('−', '')}
                    </td>
                    <td>{tx.method ? <PayChip p={tx.method} /> : <span className="faint">—</span>}</td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {tx.date}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination
            page={meta?.page ?? 1}
            pages={meta?.pages ?? 1}
            total={meta?.total ?? rows.length}
            shown={rows.length}
            limit={meta?.limit}
            label={t('pg_transactions')}
            onPage={setPage}
          />
        </>
      )}
    </div>
  )
}

/** The revenue summary (KPIs + the three breakdown tables) as CSV/PDF rows. */
function revenueRows(rev: RevenueSummary, t: TFunction): (string | number)[][] {
  return [
    [t('fin_metric'), t('fin_value')],
    [t('fin_kpi_total_revenue'), rev.totalRevenue.toFixed(2)],
    [t('fin_kpi_this_month'), rev.monthRevenue.toFixed(2)],
    [t('fin_refund_rate_pct'), (rev.refundRate * 100).toFixed(1)],
    [t('fin_kpi_wallet_topups'), rev.walletTopups.toFixed(2)],
    [],
    [t('fin_by_method'), t('col_amount')],
    ...rev.byMethod.map((d) => [d.label, d.value.toFixed(2)]),
    [],
    [t('fin_by_category'), t('col_amount')],
    ...rev.byCategory.map((d) => [d.label, d.value.toFixed(2)]),
    [],
    [t('fin_by_currency'), t('col_amount')],
    ...rev.byCurrency.map((d) => [d.label, d.value.toFixed(2)]),
  ]
}

function exportRevenue(rev: RevenueSummary, range: string, t: TFunction): void {
  downloadCsv(`revenue-${range}-${new Date().toISOString().slice(0, 10)}.csv`, revenueRows(rev, t))
}

function exportRevenuePdf(rev: RevenueSummary, range: string, t: TFunction): void {
  const date = new Date().toISOString().slice(0, 10)
  void downloadPdf(
    `revenue-${range}-${date}.pdf`,
    t('fin_pdf_title', { range: t(`fin_range_${range}`, { defaultValue: range }), date }),
    [t('fin_metric'), t('fin_value')],
    revenueRows(rev, t),
  )
}

/** Convert absolute revenue buckets to display percentages (whole numbers). */
function toPct(items: LabelValue[]): { label: string; value: number; raw: number }[] {
  const total = items.reduce((s, d) => s + d.value, 0) || 1
  return items.map((d) => ({ label: d.label, value: Math.round((d.value / total) * 100), raw: d.value }))
}

/** KPIs + charts derived from completed orders and the ledger top-ups total. */
function RevenueTab() {
  const { t } = useTranslation()
  const [range, setRange] = useState<Range>('monthly')
  const { data, isLoading, isError, refetch } = useRevenueSummary(range)
  const rev = data?.data

  if (isLoading) return <LoadingSpinner />
  if (isError || !rev) return <ErrorState message={t('fin_rev_load_error')} onRetry={() => refetch()} />

  const methodData = rev.byMethod.map((d) => {
    const meta = METHOD_META[d.label]
    return { label: meta ? t(meta.labelKey) : d.label, value: d.value, color: meta?.color ?? '#888' }
  })
  const methodPct = toPct(rev.byMethod)
  const categoryPct = toPct(rev.byCategory)
  const currencyTotal = rev.byCurrency.reduce((s, d) => s + d.value, 0) || 1

  const kpis: [string, string][] = [
    [t('fin_kpi_total_revenue'), money(rev.totalRevenue)],
    [t('fin_kpi_this_month'), money(rev.monthRevenue)],
    [t('fin_kpi_refund_rate'), (rev.refundRate * 100).toFixed(1) + '%'],
    [t('fin_kpi_wallet_topups'), money(rev.walletTopups)],
  ]

  return (
    <div>
      <div className="kpigrid" style={{ gridTemplateColumns: 'repeat(4,1fr)' }}>
        {kpis.map(([l, v]) => (
          <div className="kpi" key={l}>
            <div className="k-top">
              <div className="k-ic" style={{ background: 'var(--grad-soft)', color: 'var(--brand-1)' }}>
                <Icon name="coins" size={16} />
              </div>
              <div className="k-label">{l}</div>
            </div>
            <div className="k-val" style={{ fontSize: 23 }}>
              {v}
            </div>
          </div>
        ))}
      </div>
      <div className="dash-2col mb16">
        <div className="acard">
          <div className="panelhead">
            <Icon name="activity" size={17} />
            <h3>{t('fin_rev_over_time')}</h3>
            <div className="ph-act">
              <div className="aseg">
                {(['daily', 'weekly', 'monthly'] as Range[]).map((k) => (
                  <button key={k} className={range === k ? 'on' : ''} onClick={() => setRange(k)}>
                    {t(k)}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <div style={{ padding: '18px 16px 8px' }}>
            <AreaChart data={rev.series} labels={rev.labels} height={220} />
          </div>
        </div>
        <div className="acard">
          <div className="panelhead">
            <Icon name="card" size={17} />
            <h3>{t('fin_by_payment_method')}</h3>
          </div>
          <div style={{ padding: 22 }}>
            {methodData.length === 0 ? (
              <EmptyState title={t('fin_no_revenue')} />
            ) : (
              <div className="donut-wrap" style={{ marginBottom: 18 }}>
                <Donut
                  data={methodData}
                  size={140}
                  thickness={20}
                  center={
                    <div>
                      <div className="num" style={{ fontSize: 18, fontWeight: 800 }}>
                        {money(rev.totalRevenue)}
                      </div>
                      <div style={{ fontSize: 10.5, color: 'var(--text-faint)', fontWeight: 700 }}>{t('fin_all_time')}</div>
                    </div>
                  }
                />
                <div className="legend" style={{ flex: 1 }}>
                  {methodData.map((d, i) => (
                    <div className="li" key={d.label}>
                      <span className="sw" style={{ background: d.color }} />
                      <span style={{ fontWeight: 700 }}>{d.label}</span>
                      <span className="lv num">{methodPct[i].value}%</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
      <div className="g2">
        <div className="acard pad">
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>{t('fin_rev_by_category')}</h3>
          {categoryPct.length === 0 ? (
            <EmptyState title={t('fin_no_revenue')} />
          ) : (
            <Bars data={categoryPct.map((c) => ({ label: c.label, value: c.value, color: 'var(--grad)' }))} />
          )}
        </div>
        <div className="acard pad">
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>{t('fin_by_currency')}</h3>
          <div className="deflist">
            {rev.byCurrency.length === 0 ? (
              <div className="faint" style={{ fontSize: 13 }}>
                {t('fin_no_revenue')}
              </div>
            ) : (
              rev.byCurrency.map((c) => (
                <div className="defrow" key={c.label}>
                  <span className="dk">{t('fin_currency_revenue', { currency: c.label })}</span>
                  <span className="dv num">
                    {money(c.value, c.label === 'TRY' ? 'TRY' : 'USD')} ·{' '}
                    {Math.round((c.value / currencyTotal) * 100)}%
                  </span>
                </div>
              ))
            )}
          </div>
          <div style={{ marginTop: 16, display: 'flex', gap: 10 }}>
            <button className="abtn sm" onClick={() => exportRevenue(rev, range, t)}>
              <Icon name="download" size={14} /> {t('fin_export_csv')}
            </button>
            <button className="abtn sm" onClick={() => exportRevenuePdf(rev, range, t)}>
              <Icon name="file" size={14} /> {t('fin_export_pdf')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const USDT_STATUS_CLASS: Record<TopUpStatus, string> = {
  pending: 'st st-warn',
  approved: 'st st-ok',
  rejected: 'st st-danger',
}

const USDT_FILTERS: [TopUpStatus | '', string][] = [
  ['pending', 'top_filter_pending'],
  ['approved', 'top_filter_approved'],
  ['rejected', 'top_filter_rejected'],
  ['', 'all'],
]

/** USDT verification queue — the top-up request queue filtered to the `usdt`
 *  channel. Reuses the wired approve-and-credit / reject-with-reason endpoints;
 *  the customer's note carries the on-chain reference. */
function UsdtQueueTab() {
  const { t } = useTranslation()
  // The USDT queue reuses the top-up decision endpoints, so it follows topups.manage.
  const canManage = useCan()('topups.manage')
  const [status, setStatus] = useState<TopUpStatus | ''>('pending')
  const [page, setPage] = useState(1)
  const [toReject, setToReject] = useState<AdminTopUp | null>(null)

  const { data, isLoading, isError, refetch } = useTopUps({ page, status: status || undefined, channel: 'usdt' })
  const rows = data?.data ?? []
  const meta = data?.meta

  const approveM = useApproveTopUp()
  const [approveError, setApproveError] = useState('')
  const approve = (id: string) => {
    setApproveError('')
    approveM.mutate(id, { onError: (e) => setApproveError(apiErrorText(e, 'top_approve_failed')) })
  }

  return (
    <div className="acard">
      <div className="toolbar">
        <div className="chiprow">
          {USDT_FILTERS.map(([k, l]) => (
            <Chip
              key={k || 'all'}
              on={status === k}
              onClick={() => {
                setStatus(k)
                setPage(1)
              }}
            >
              {t(l)}
            </Chip>
          ))}
        </div>
        <button className="abtn" style={{ marginInlineStart: 'auto' }} onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> {t('top_refresh')}
        </button>
      </div>

      {approveError && <div style={{ padding: '10px 18px', color: 'var(--danger)', fontSize: 13 }}>{approveError}</div>}

      {isLoading ? (
        <LoadingSpinner />
      ) : isError && !data ? (
        <ErrorState message={t('fin_usdt_load_error')} onRetry={() => refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState title={t('fin_usdt_empty')} sub={t('fin_usdt_empty_sub')} />
      ) : (
        <>
          <div>
            {rows.map((r) => (
              <div key={r.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                  <b style={{ fontSize: 15 }}>{money(r.amount)}</b>
                  <span className="bdg">{t(`top_channel_${r.channel}_name`, { defaultValue: r.channel.toUpperCase() })}</span>
                  <span className="faint" style={{ fontSize: 12.5 }}>
                    {r.customerEmail || r.customerPhone || r.userId.slice(-8)}
                  </span>
                  {r.role && <RoleBadge role={r.role} />}
                  <span className="faint" style={{ fontSize: 12 }}>
                    · {formatDateTime(r.createdAt)}
                  </span>
                  <span style={{ marginInlineStart: 'auto' }} className={USDT_STATUS_CLASS[r.status]}>
                    <i className="d" />
                    {t(`top_status_${r.status}`, { defaultValue: r.status })}
                  </span>
                </div>
                {r.note && (
                  <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>{t('fin_tx_reference', { note: r.note })}</div>
                )}
                {r.status === 'rejected' && r.decisionReason && (
                  <div style={{ fontSize: 12.5, color: 'var(--danger)', marginTop: 6 }}>{t('top_rejected_reason', { reason: r.decisionReason })}</div>
                )}
                {r.status !== 'pending' && r.decidedBy && (
                  <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>
                    {t('top_decided_by', { name: r.decidedBy })}
                    {r.decidedAt ? ` · ${formatDateTime(r.decidedAt)}` : ''}
                  </div>
                )}
                {canManage && r.status === 'pending' && (
                  <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
                    <button className="abtn xs ok" disabled={approveM.isPending} onClick={() => approve(r.id)}>
                      <Icon name="check" size={13} /> {t('fin_verify_credit')}
                    </button>
                    <button className="abtn xs danger" disabled={approveM.isPending} onClick={() => setToReject(r)}>
                      <Icon name="x" size={13} /> {t('reject')}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
          <Pagination
            page={meta?.page ?? 1}
            pages={meta?.pages ?? 1}
            total={meta?.total ?? rows.length}
            shown={rows.length}
            limit={meta?.limit}
            label={t('pg_requests')}
            onPage={setPage}
          />
        </>
      )}

      {toReject && <UsdtRejectModal req={toReject} onClose={() => setToReject(null)} />}
    </div>
  )
}

/** Reject a USDT top-up request with a required reason (shown to the customer). */
function UsdtRejectModal({ req, onClose }: { req: AdminTopUp; onClose: () => void }) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const rejectM = useRejectTopUp()

  const apply = () => {
    if (!reason.trim()) {
      setError(t('top_reason_required'))
      return
    }
    setError('')
    rejectM.mutate(
      { id: req.id, reason: reason.trim() },
      { onSuccess: onClose, onError: (e) => setError(apiErrorText(e, 'top_reject_failed')) },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>{t('fin_reject_usdt_title')}</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          <Trans
            i18nKey="top_reject_body"
            values={{ amount: money(req.amount), who: req.customerEmail || t('top_customer') }}
            components={{ b: <b /> }}
          />
        </p>
        <label className="alabel">{t('top_reason_label')}</label>
        <input
          className="afield"
          placeholder={t('fin_reject_ph')}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <div style={{ color: 'var(--danger)', fontSize: 12.5, marginTop: 10 }}>{error}</div>}
        <div style={{ display: 'flex', gap: 10, marginTop: 18, justifyContent: 'flex-end' }}>
          <button className="abtn" onClick={onClose} disabled={rejectM.isPending}>
            {t('cancel')}
          </button>
          <button className="abtn danger" onClick={apply} disabled={rejectM.isPending}>
            <Icon name="x" size={15} /> {t('reject')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
