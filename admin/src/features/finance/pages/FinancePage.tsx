import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import {
  Icon,
  PageHead,
  Avatar,
  PayChip,
  AreaChart,
  Donut,
  Bars,
  Pagination,
  Tabs,
  ComingSoonNote,
  LoadingSpinner,
  ErrorState,
  EmptyState,
} from '@/components'
import { money } from '@/lib/utils'
import { useTransactions, useRevenueSummary } from '../hooks/useFinance'
import { adaptTx } from '../lib/adaptFinance'
import type { TxType, LabelValue } from '../api/finance'

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

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_wallet')]}
        title="Wallet & financial management"
        sub="Money movement across the platform wallet ledger"
      >
        <button className="abtn" disabled title="Coming soon">
          <Icon name="download" size={15} /> {t('export')}
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
      {tab === 'usdt' && (
        <ComingSoonNote
          mock
          note="USDT top-ups auto-confirm on a network confirmation today, so there is no pending queue. A manual verification queue will land with the USDT-pending payment flow."
        />
      )}
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
            label="transactions"
            onPage={setPage}
          />
        </>
      )}
    </div>
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
            {/* TODO: implement real CSV/PDF export endpoints. */}
            <button className="abtn sm" disabled title="Coming soon">
              <Icon name="download" size={14} /> Export CSV
            </button>
            <button className="abtn sm" disabled title="Coming soon">
              <Icon name="file" size={14} /> Export PDF
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
