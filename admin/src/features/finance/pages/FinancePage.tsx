import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Avatar, PayChip, AreaChart, Donut, Bars, Pagination, Tabs, ComingSoonNote } from '@/components'
import { money } from '@/lib/utils'
import { transactions, revSeries, revLabels, payBreakdown, catRevenue, usdtQueue } from '@/lib/mock/demo'

type Tab = 'transactions' | 'revenue' | 'usdt'
type Range = 'daily' | 'weekly' | 'monthly'

const TX_COLOR: Record<string, string> = {
  topup: 'var(--ok)',
  purchase: 'var(--ff-code)',
  refund: 'var(--warn)',
  adjust: 'var(--brand-2)',
  'sub-balance': 'var(--agent)',
}

export default function FinancePage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('transactions')
  const [range, setRange] = useState<Range>('monthly')

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('grp_finance'), t('nav_wallet')]}
        title="Wallet & financial management"
        sub="Unified money movement across the platform"
      >
        <button className="abtn">
          <Icon name="download" size={15} /> {t('export')}
        </button>
      </PageHead>

      <ComingSoonNote mock />

      <Tabs<Tab>
        value={tab}
        onChange={setTab}
        items={[
          { k: 'transactions', label: 'All transactions' },
          { k: 'revenue', label: 'Revenue summary' },
          {
            k: 'usdt',
            label: (
              <>
                USDT queue
                <span className="sb-badge" style={{ display: 'inline-grid', marginInlineStart: 7, position: 'static' }}>
                  4
                </span>
              </>
            ),
          },
        ]}
      />

      {tab === 'transactions' && (
        <div className="acard">
          <div className="toolbar">
            <div className="fsearch">
              <Icon name="search" size={15} />
              <input placeholder="Search by user or transaction ID…" />
            </div>
            <select className="select">
              <option>All types</option>
              <option>Top-up</option>
              <option>Purchase</option>
              <option>Refund</option>
              <option>Adjustment</option>
            </select>
            <select className="select">
              <option>All methods</option>
              <option>Wallet</option>
              <option>Visa</option>
              <option>USDT</option>
            </select>
            <div className="tb-spacer" />
            <button className="abtn sm">
              <Icon name="filter" size={14} /> Date range
            </button>
          </div>
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
                {transactions.map((tx) => (
                  <tr key={tx.id}>
                    <td className="mono strong">{tx.id}</td>
                    <td>
                      <div className="cellprod">
                        <Avatar name={tx.user} />
                        <div className="pn">
                          <b style={{ fontSize: 12.5 }}>{tx.user}</b>
                          <span className={'pill-role role-' + tx.role} style={{ fontSize: 10 }}>
                            {tx.role}
                          </span>
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
                      {money(Math.abs(tx.amount), tx.cur).replace('−', '')}
                    </td>
                    <td>{tx.method === '—' ? <span className="faint">—</span> : <PayChip p={tx.method} />}</td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {tx.date}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination total={48230} pages={5} label="transactions" />
        </div>
      )}

      {tab === 'revenue' && (
        <div>
          <div className="kpigrid" style={{ gridTemplateColumns: 'repeat(4,1fr)' }}>
            {(
              [
                ['Total revenue', '$1.91M', '14.2%', 'up'],
                ['This month', '$240K', '12.1%', 'up'],
                ['Refund rate', '1.4%', '0.3%', 'down'],
                ['Net margin', '$182K', '9.6%', 'up'],
              ] as [string, string, string, 'up' | 'down'][]
            ).map(([l, v, d, dir]) => (
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
                <div className="k-foot">
                  <span className={'delta ' + dir}>
                    <Icon name={dir === 'up' ? 'arrowup' : 'arrowdown'} size={12} stroke={2.6} />
                    {d}
                  </span>
                  <span className="k-since">vs last period</span>
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
                <AreaChart data={revSeries[range]} labels={revLabels[range]} height={220} />
              </div>
            </div>
            <div className="acard">
              <div className="panelhead">
                <Icon name="card" size={17} />
                <h3>By payment method</h3>
              </div>
              <div style={{ padding: 22 }}>
                <div className="donut-wrap" style={{ marginBottom: 18 }}>
                  <Donut
                    data={payBreakdown}
                    size={140}
                    thickness={20}
                    center={
                      <div>
                        <div className="num" style={{ fontSize: 18, fontWeight: 800 }}>
                          $240K
                        </div>
                        <div style={{ fontSize: 10.5, color: 'var(--text-faint)', fontWeight: 700 }}>this month</div>
                      </div>
                    }
                  />
                  <div className="legend" style={{ flex: 1 }}>
                    {payBreakdown.map((d) => (
                      <div className="li" key={d.label}>
                        <span className="sw" style={{ background: d.color }} />
                        <span style={{ fontWeight: 700 }}>{d.label}</span>
                        <span className="lv num">{d.value}%</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="g2">
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>Revenue by category</h3>
              <Bars data={catRevenue.map((c) => ({ ...c, color: 'var(--grad)' }))} />
            </div>
            <div className="acard pad">
              <h3 style={{ fontSize: 15, fontWeight: 800, marginBottom: 16 }}>By currency</h3>
              <div className="deflist">
                <div className="defrow">
                  <span className="dk">USD revenue</span>
                  <span className="dv num">$198,400 · 83%</span>
                </div>
                <div className="defrow">
                  <span className="dk">TRY revenue</span>
                  <span className="dv num">₺1,348,200 · 17%</span>
                </div>
                <div className="defrow">
                  <span className="dk">USDT settled</span>
                  <span className="dv num">$31,200</span>
                </div>
              </div>
              <div style={{ marginTop: 16, display: 'flex', gap: 10 }}>
                {/* TODO: implement real CSV/PDF export. */}
                <button className="abtn sm">
                  <Icon name="download" size={14} /> Export CSV
                </button>
                <button className="abtn sm">
                  <Icon name="file" size={14} /> Export PDF
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === 'usdt' && (
        <div className="acard">
          <div className="panelhead">
            <span
              className="bdg"
              style={{ background: 'rgba(34,227,200,.16)', color: 'var(--accent)', border: '1px solid rgba(34,227,200,.34)' }}
            >
              <i className="d" style={{ background: 'var(--accent)' }} />
              USDT
            </span>
            <h3>Pending USDT confirmations</h3>
            <span className="faint" style={{ fontSize: 12.5, marginInlineStart: 6 }}>
              Optional manual verification queue
            </span>
          </div>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Transaction</th>
                  <th>User</th>
                  <th>Amount</th>
                  <th>Network</th>
                  <th>Tx hash</th>
                  <th>Submitted</th>
                  <th style={{ textAlign: 'end' }}>Action</th>
                </tr>
              </thead>
              <tbody>
                {usdtQueue.map((q) => (
                  <tr key={q.id}>
                    <td className="mono strong">{q.id}</td>
                    <td>
                      <div className="cellprod">
                        <Avatar name={q.user} />
                        <div className="pn">
                          <b style={{ fontSize: 12.5 }}>{q.user}</b>
                        </div>
                      </div>
                    </td>
                    <td className="num strong">${q.amount.toFixed(2)}</td>
                    <td>
                      <span className="st st-info">
                        <i className="d" />
                        {q.network}
                      </span>
                    </td>
                    <td className="mono muted">{q.txhash}</td>
                    <td className="muted" style={{ fontSize: 12 }}>
                      {q.submitted}
                    </td>
                    <td>
                      <div className="row-actions">
                        <button className="abtn xs ok">
                          <Icon name="check" size={13} /> Verify
                        </button>
                        <button className="abtn xs danger">
                          <Icon name="x" size={13} /> Reject
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div
            style={{
              padding: 16,
              display: 'flex',
              gap: 8,
              alignItems: 'center',
              color: 'var(--text-dim)',
              fontSize: 12.5,
              borderTop: '1px solid var(--border)',
            }}
          >
            <Icon name="alert" size={15} /> Open item: USDT top-ups may be auto-confirmed on 1 network confirmation. This
            manual queue is optional.
          </div>
        </div>
      )}
    </div>
  )
}
