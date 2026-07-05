import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
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
import { money, downloadCsv, downloadPdf } from '@/lib/utils'
import { ApiError } from '@/lib/api-client'
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
const METHOD_META: Record<string, { label: string; color: string }> = {
  wallet: { label: 'Store wallet', color: '#8a3bff' },
  card: { label: 'Visa / Mastercard', color: '#3b5bff' },
  usdt: { label: 'USDT', color: '#22e3c8' },
}

export default function FinancePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('transactions')
  const [exporting, setExporting] = useState(false)

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

      const header = ['Transaction ID', 'User', 'Role', 'Type', 'Amount', 'Balance after', 'Method', 'Ref', 'Date']
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
        crumbs={[t('grp_finance'), t('nav_wallet')]}
        title="Wallet & financial management"
        sub="Money movement across the platform wallet ledger"
      >
        <button className="abtn" onClick={handleExport} disabled={exporting}>
          <Icon name="download" size={15} /> {exporting ? '…' : t('export')}
        </button>
      </PageHead>

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'transactions', label: 'All transactions' },
          { k: 'revenue', label: 'Revenue summary' },
          { k: 'usdt', label: 'USDT queue' },
        ]}
      />

      {tab === 'transactions' && <TransactionsTab />}
      {tab === 'revenue' && <RevenueTab />}
      {tab === 'usdt' && <UsdtQueueTab />}
    </div>
  )
}

/** The wallet-ledger feed: filter by type/method, search by user, paginate. */
function TransactionsTab() {
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
            placeholder="Search by user or transaction ID…"
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
          <option value="">All types</option>
          <option value="topup">Top-up</option>
          <option value="purchase">Purchase</option>
          <option value="refund">Refund</option>
          <option value="adjustment">Adjustment</option>
        </select>
        <select
          className="select"
          value={method}
          onChange={(e) => {
            setMethod(e.target.value)
            setPage(1)
          }}
        >
          <option value="">All methods</option>
          <option value="wallet">Wallet</option>
          <option value="card">Card</option>
          <option value="usdt">USDT</option>
        </select>
      </div>

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Couldn't load transactions." onRetry={() => refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState title="No transactions found" />
      ) : (
        <>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>User</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>Method</th>
                  <th>Date</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((tx) => (
                  <tr key={tx.id}>
                    <td className="mono strong">{tx.id.slice(-8)}</td>
                    <td>
                      <div className="cellprod">
                        <Avatar name={tx.user} />
                        <div className="pn">
                          <b style={{ fontSize: 12.5 }}>{tx.user}</b>
                          {tx.role && (
                            <span className={'pill-role role-' + tx.role} style={{ fontSize: 10 }}>
                              {tx.role}
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
                        {tx.type}
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
function revenueRows(rev: RevenueSummary): (string | number)[][] {
  return [
    ['Metric', 'Value'],
    ['Total revenue', rev.totalRevenue.toFixed(2)],
    ['This month', rev.monthRevenue.toFixed(2)],
    ['Refund rate %', (rev.refundRate * 100).toFixed(1)],
    ['Wallet top-ups', rev.walletTopups.toFixed(2)],
    [],
    ['By method', 'Amount'],
    ...rev.byMethod.map((d) => [d.label, d.value.toFixed(2)]),
    [],
    ['By category', 'Amount'],
    ...rev.byCategory.map((d) => [d.label, d.value.toFixed(2)]),
    [],
    ['By currency', 'Amount'],
    ...rev.byCurrency.map((d) => [d.label, d.value.toFixed(2)]),
  ]
}

function exportRevenue(rev: RevenueSummary, range: string): void {
  downloadCsv(`revenue-${range}-${new Date().toISOString().slice(0, 10)}.csv`, revenueRows(rev))
}

function exportRevenuePdf(rev: RevenueSummary, range: string): void {
  const date = new Date().toISOString().slice(0, 10)
  void downloadPdf(`revenue-${range}-${date}.pdf`, `Revenue summary (${range}) — ${date}`, ['Metric', 'Value'], revenueRows(rev))
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
  if (isError || !rev) return <ErrorState message="Couldn't load revenue summary." onRetry={() => refetch()} />

  const methodData = rev.byMethod.map((d) => {
    const meta = METHOD_META[d.label] ?? { label: d.label, color: '#888' }
    return { label: meta.label, value: d.value, color: meta.color }
  })
  const methodPct = toPct(rev.byMethod)
  const categoryPct = toPct(rev.byCategory)
  const currencyTotal = rev.byCurrency.reduce((s, d) => s + d.value, 0) || 1

  const kpis: [string, string][] = [
    ['Total revenue', money(rev.totalRevenue)],
    ['This month', money(rev.monthRevenue)],
    ['Refund rate', (rev.refundRate * 100).toFixed(1) + '%'],
    ['Wallet top-ups', money(rev.walletTopups)],
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
            <h3>Revenue over time</h3>
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
            <h3>By payment method</h3>
          </div>
          <div style={{ padding: 22 }}>
            {methodData.length === 0 ? (
              <EmptyState title="No revenue yet" />
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
                      <div style={{ fontSize: 10.5, color: 'var(--text-faint)', fontWeight: 700 }}>all time</div>
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
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>Revenue by category</h3>
          {categoryPct.length === 0 ? (
            <EmptyState title="No revenue yet" />
          ) : (
            <Bars data={categoryPct.map((c) => ({ label: c.label, value: c.value, color: 'var(--grad)' }))} />
          )}
        </div>
        <div className="acard pad">
          <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>By currency</h3>
          <div className="deflist">
            {rev.byCurrency.length === 0 ? (
              <div className="faint" style={{ fontSize: 13 }}>
                No revenue yet
              </div>
            ) : (
              rev.byCurrency.map((c) => (
                <div className="defrow" key={c.label}>
                  <span className="dk">{c.label} revenue</span>
                  <span className="dv num">
                    {money(c.value, c.label === 'TRY' ? 'TRY' : 'USD')} ·{' '}
                    {Math.round((c.value / currencyTotal) * 100)}%
                  </span>
                </div>
              ))
            )}
          </div>
          <div style={{ marginTop: 16, display: 'flex', gap: 10 }}>
            <button className="abtn sm" onClick={() => exportRevenue(rev, range)}>
              <Icon name="download" size={14} /> Export CSV
            </button>
            <button className="abtn sm" onClick={() => exportRevenuePdf(rev, range)}>
              <Icon name="file" size={14} /> Export PDF
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
  ['pending', 'Pending'],
  ['approved', 'Approved'],
  ['rejected', 'Rejected'],
  ['', 'All'],
]

/** USDT verification queue — the top-up request queue filtered to the `usdt`
 *  channel. Reuses the wired approve-and-credit / reject-with-reason endpoints;
 *  the customer's note carries the on-chain reference. */
function UsdtQueueTab() {
  const { t } = useTranslation()
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
    approveM.mutate(id, { onError: (e) => setApproveError(e instanceof ApiError ? e.message : 'Approval failed.') })
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
              {l}
            </Chip>
          ))}
        </div>
        <button className="abtn" style={{ marginInlineStart: 'auto' }} onClick={() => refetch()}>
          <Icon name="refresh" size={15} /> Refresh
        </button>
      </div>

      {approveError && <div style={{ padding: '10px 18px', color: 'var(--danger)', fontSize: 13 }}>{approveError}</div>}

      {isLoading ? (
        <LoadingSpinner />
      ) : isError ? (
        <ErrorState message="Couldn't load the USDT queue." onRetry={() => refetch()} />
      ) : rows.length === 0 ? (
        <EmptyState
          title="No USDT requests"
          sub="USDT wallet-funding requests appear here for verification once a customer declares an on-chain payment."
        />
      ) : (
        <>
          <div>
            {rows.map((r) => (
              <div key={r.id} style={{ borderBottom: '1px solid var(--border)', padding: '14px 18px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
                  <b style={{ fontSize: 15 }}>{money(r.amount)}</b>
                  <span className="bdg">{r.channel.toUpperCase()}</span>
                  <span className="faint" style={{ fontSize: 12.5 }}>
                    {r.customerEmail || r.customerPhone || r.userId.slice(-8)}
                  </span>
                  {r.role && <RoleBadge role={r.role} />}
                  <span className="faint" style={{ fontSize: 12 }}>
                    · {new Date(r.createdAt).toLocaleString()}
                  </span>
                  <span style={{ marginInlineStart: 'auto' }} className={USDT_STATUS_CLASS[r.status]}>
                    <i className="d" />
                    {r.status}
                  </span>
                </div>
                {r.note && (
                  <div style={{ fontSize: 13, color: 'var(--text-dim)', marginTop: 6 }}>Tx reference: {r.note}</div>
                )}
                {r.status === 'rejected' && r.decisionReason && (
                  <div style={{ fontSize: 12.5, color: 'var(--danger)', marginTop: 6 }}>Rejected: {r.decisionReason}</div>
                )}
                {r.status !== 'pending' && r.decidedBy && (
                  <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>
                    Decided by {r.decidedBy}
                    {r.decidedAt ? ` · ${new Date(r.decidedAt).toLocaleString()}` : ''}
                  </div>
                )}
                {r.status === 'pending' && (
                  <div style={{ display: 'flex', gap: 8, marginTop: 10 }}>
                    <button className="abtn xs ok" disabled={approveM.isPending} onClick={() => approve(r.id)}>
                      <Icon name="check" size={13} /> Verify & credit
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
      setError('A rejection reason is required.')
      return
    }
    setError('')
    rejectM.mutate(
      { id: req.id, reason: reason.trim() },
      { onSuccess: onClose, onError: (e) => setError(e instanceof ApiError ? e.message : 'Rejection failed.') },
    )
  }

  return (
    <Modal onClose={onClose} maxWidth={440}>
      <div style={{ padding: 22 }}>
        <h3 style={{ fontSize: 17, fontWeight: 800, marginBottom: 6 }}>Reject USDT request</h3>
        <p style={{ fontSize: 13.5, color: 'var(--text-dim)', marginBottom: 16 }}>
          Reject the <b>{money(req.amount)}</b> request from <b>{req.customerEmail || 'customer'}</b>. The reason is
          shown to the customer.
        </p>
        <label className="alabel">Reason (required)</label>
        <input
          className="afield"
          placeholder="e.g. no matching on-chain payment found…"
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
