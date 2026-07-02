import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { Icon, PageHead, Spark, AreaChart, Donut, Avatar, FfBadge, StatusBadge, Art, artForCategory } from '@/components'
import type { DonutDatum } from '@/components'
import { money, downloadCsv, formatLongDate } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'
import { useDashboardStats, useLowStock, useRevenueChart, useFulfillmentBreakdown, useHealth } from '../hooks/useDashboard'
import { useOrders } from '@/features/orders/hooks/useOrders'
import { adaptOrder } from '@/features/orders/lib/adaptOrder'

type Range = 'daily' | 'weekly' | 'monthly'

// Time-of-day greeting for the dashboard header.
function greeting(h: number): string {
  if (h < 12) return 'Good morning'
  if (h < 18) return 'Good afternoon'
  return 'Good evening'
}

// Donut colors keyed by fulfillment slice key — the design-system tokens (same
// source as FfBadge / OrderListPage's FF_DOT), so the palette tracks the theme.
const FF_COLOR: Record<string, string> = {
  code: 'var(--ff-code)',
  credit: 'var(--ff-credit)',
  transfer: 'var(--ff-transfer)',
}

interface KpiProps {
  icon: React.ReactNode
  iconBg: string
  label: string
  value: string
  delta?: string
  deltaDir?: 'up' | 'down' | 'flat'
  since?: string
  spark?: number[]
  sparkColor?: string
  cls?: string
  onClick?: () => void
}

function Kpi({ icon, iconBg, label, value, delta, deltaDir, since, spark, sparkColor, cls, onClick }: KpiProps) {
  return (
    <div
      className={'kpi' + (cls ? ' ' + cls : '')}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={onClick ? (e) => (e.key === 'Enter' || e.key === ' ') && onClick() : undefined}
      style={onClick ? { cursor: 'pointer' } : undefined}
    >
      <div className="k-top">
        <div className="k-ic" style={{ background: iconBg }}>
          {icon}
        </div>
        <div className="k-label">{label}</div>
      </div>
      <div className="k-val">{value}</div>
      <div className="k-foot">
        {delta != null && (
          <span className={'delta ' + deltaDir}>
            <Icon name={deltaDir === 'up' ? 'arrowup' : deltaDir === 'down' ? 'arrowdown' : 'minus'} size={12} stroke={2.6} />
            {delta}
          </span>
        )}
        {since && <span className="k-since">{since}</span>}
      </div>
      {spark && (
        <div style={{ position: 'absolute', inset: 'auto 0 0 0', opacity: 0.5 }}>
          <Spark data={spark} color={sparkColor} h={30} fill />
        </div>
      )}
    </div>
  )
}

export default function DashboardPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const user = useAuthStore((st) => st.user)
  const [range, setRange] = useState<Range>('daily')
  const { data: statsRes } = useDashboardStats()
  const { data: lowRes } = useLowStock()
  const { data: revRes } = useRevenueChart(range)
  const { data: ffRes } = useFulfillmentBreakdown()
  const { data: recentRes } = useOrders({ limit: 7 })
  const { data: health } = useHealth()
  const s = statsRes?.data

  // Revenue area chart — real completed-order revenue for the selected range.
  const rev = revRes?.data?.series ?? []
  const lab = revRes?.data?.labels ?? []
  const revTotal = rev.reduce((a, n) => a + n, 0)

  // Fulfillment donut — real line-item counts by fulfillment type.
  const ffSlices: DonutDatum[] = (ffRes?.data ?? []).map((d) => ({
    key: d.key,
    label: d.label,
    value: d.value,
    color: FF_COLOR[d.key] ?? '#888',
  }))
  const ffTotal = ffSlices.reduce((a, d) => a + d.value, 0)

  // Recent orders — newest 7, adapted to the row view.
  const recent = (recentRes?.data ?? []).map(adaptOrder)

  // Low-stock panel — real inventory, newest threshold breaches first.
  const lowStock = lowRes?.data ?? []

  // Export the current dashboard view (KPI summary + recent orders) to CSV.
  const handleExport = () => {
    const rows: (string | number)[][] = [
      ['SalehCard dashboard export', formatLongDate()],
      [],
      ['Metric', 'Value'],
      ["Today's revenue", s ? s.revenueToday : ''],
      ["Today's orders", s ? s.ordersToday : ''],
      ['Active users (24h)', s ? s.activeUsers : ''],
      ['Wallet top-ups (today)', s ? s.walletTopups : ''],
      ['Pending transfers', s ? s.pendingTransfers : ''],
      ['Low-stock alerts', s ? s.lowStockCount : ''],
      [],
      ['Recent orders'],
      ['Order', 'Customer', 'Type', 'Amount', 'Currency', 'Status', 'Date'],
      ...recent.map((o) => [o.id, o.customer, o.ff, o.amount, o.cur, o.status, o.date]),
    ]
    downloadCsv(`dashboard-${new Date().toISOString().slice(0, 10)}.csv`, rows)
  }

  return (
    <div className="page page-wide">
      <PageHead
        crumbs={[t('nav_dashboard')]}
        title={`${greeting(new Date().getHours())}, ${user?.name ?? 'Admin'}`}
        sub={`Here's what's happening across SalehCard today — ${formatLongDate()}.`}
      >
        <button className="abtn" onClick={handleExport}>
          <Icon name="download" size={15} /> {t('export')}
        </button>
        <button className="abtn primary" onClick={() => navigate('/products/new')}>
          <Icon name="plus" size={15} /> {t('new_product')}
        </button>
      </PageHead>

      <div className="kpigrid">
        <Kpi
          icon={<Icon name="wallet" size={17} />}
          iconBg="var(--grad)"
          label="Today's revenue"
          value={s ? money(s.revenueToday) : '—'}
          delta={s ? `${s.revenueDeltaPct}%` : undefined}
          deltaDir={s && s.revenueDeltaPct < 0 ? 'down' : 'up'}
          since="vs yesterday"
          spark={s?.revenueSpark}
          sparkColor="#8a3bff"
        />
        <Kpi
          icon={<Icon name="bag" size={17} />}
          iconBg="linear-gradient(135deg,#3b5bff,#22e3c8)"
          label="Today's orders"
          value={s ? s.ordersToday.toLocaleString() : '—'}
          delta={s ? `${s.ordersDeltaPct}%` : undefined}
          deltaDir={s && s.ordersDeltaPct < 0 ? 'down' : 'up'}
          since="vs yesterday"
          spark={s?.ordersSpark}
          sparkColor="#3b5bff"
        />
        <Kpi
          icon={<Icon name="users" size={17} />}
          iconBg="linear-gradient(135deg,#2fd47a,#22e3c8)"
          label="Active users"
          value={s ? s.activeUsers.toLocaleString() : '—'}
          since="last 24h"
        />
        <Kpi
          icon={<Icon name="coins" size={17} />}
          iconBg="linear-gradient(135deg,#8a3bff,#d633ff)"
          label="Wallet top-ups"
          value={s ? money(s.walletTopups) : '—'}
          since="today"
        />
        <Kpi
          icon={<Icon name="send" size={17} />}
          iconBg="rgba(255,155,61,.2)"
          label="Pending transfers"
          value={s ? String(s.pendingTransfers) : '—'}
          since="needs action"
          cls="alert"
        />
        <Kpi
          icon={<Icon name="coins" size={17} />}
          iconBg="rgba(255,155,61,.2)"
          label="Pending top-ups"
          value={s ? String(s.pendingTopups) : '—'}
          since="awaiting approval"
          cls="alert"
          onClick={() => navigate('/topups')}
        />
        <Kpi
          icon={<Icon name="id" size={17} />}
          iconBg="rgba(255,155,61,.2)"
          label="Pending KYC"
          value={s ? String(s.pendingKyc) : '—'}
          since="awaiting review"
          cls="alert"
          onClick={() => navigate('/kyc')}
        />
        <Kpi
          icon={<Icon name="alert" size={17} />}
          iconBg="rgba(255,77,109,.2)"
          label="Low-stock alerts"
          value={s ? String(s.lowStockCount) : '—'}
          since="codes running out"
          cls="crit"
        />
      </div>

      <div className="dash-2col mb16">
        <div className="acard">
          <div className="panelhead">
            <Icon name="activity" size={18} />
            <h3>{t('revenue')}</h3>
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
          <div style={{ padding: '16px 16px 8px' }}>
            <div className="row" style={{ gap: 22, marginBottom: 4, paddingInline: 6 }}>
              <div>
                <div style={{ fontSize: 12, color: 'var(--text-dim)', fontWeight: 700 }}>
                  {range === 'daily' ? 'Last 14 days' : range === 'weekly' ? 'Last 12 weeks' : 'Last 12 months'}
                </div>
                <div className="num" style={{ fontSize: 26, fontWeight: 800, letterSpacing: '-.02em' }}>
                  {money(revTotal)}
                </div>
              </div>
              <div className="spacer" />
              <span className="k-since" style={{ fontSize: 12 }}>
                Completed orders only
              </span>
            </div>
            <AreaChart data={rev} labels={lab} height={210} />
          </div>
        </div>

        <div className="acard">
          <div className="panelhead">
            <Icon name="pkg" size={18} />
            <h3>Orders by fulfillment</h3>
          </div>
          <div style={{ padding: 22 }}>
            <div className="donut-wrap">
              <Donut
                data={ffSlices}
                size={148}
                thickness={22}
                center={
                  <div>
                    <div className="num" style={{ fontSize: 23, fontWeight: 800 }}>
                      {ffTotal}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--text-faint)', fontWeight: 700 }}>items</div>
                  </div>
                }
              />
              <div className="legend" style={{ flex: 1 }}>
                {ffSlices.map((d) => (
                  <div className="li" key={d.key}>
                    <span className="sw" style={{ background: d.color }} />
                    <span style={{ fontWeight: 700 }}>{d.label}</span>
                    <span className="lv num">{d.value}</span>
                  </div>
                ))}
              </div>
            </div>
            <div
              style={{
                marginTop: 18,
                paddingTop: 16,
                borderTop: '1px solid var(--border)',
                fontSize: 12.5,
                color: 'var(--text-dim)',
                display: 'flex',
                gap: 7,
                alignItems: 'center',
              }}
            >
              <Icon name="bolt" size={14} /> Account-credit orders settle instantly — code orders depend on inventory.
            </div>
          </div>
        </div>
      </div>

      <div className="dash-2col">
        <div className="acard">
          <div className="panelhead">
            <Icon name="clock" size={18} />
            <h3>Recent orders</h3>
            <div className="ph-act">
              <button className="abtn xs" onClick={() => navigate('/orders')}>
                View all <Icon name="chevright" size={13} />
              </button>
            </div>
          </div>
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Customer</th>
                  <th>Type</th>
                  <th>Amount</th>
                  <th>{t('status')}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {recent.map((o) => (
                  <tr key={o.id} className="clickable" onClick={() => navigate(`/orders/${o.id}`)}>
                    <td>
                      <span className="mono strong">{o.id.slice(-8)}</span>
                      <div className="faint" style={{ fontSize: 11 }}>
                        {o.date}
                      </div>
                    </td>
                    <td>
                      <div className="cellprod">
                        <Avatar name={o.customer} />
                        <div className="pn">
                          <b>{o.customer}</b>
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
                    <td>
                      <div className="row-actions">
                        <span className="iact">
                          <Icon name="chevright" size={16} />
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div className="acard">
            <div className="panelhead">
              <Icon name="alert" size={18} />
              <h3>Low-stock alerts</h3>
              <span className="sb-badge" style={{ marginInlineStart: 6 }}>
                {s ? s.lowStockCount : lowStock.length}
              </span>
              <div className="ph-act">
                <button className="abtn xs" onClick={() => navigate('/inventory')}>
                  Manage
                </button>
              </div>
            </div>
            <div>
              {lowStock.length === 0 && (
                <div className="lsrow" style={{ color: 'var(--text-faint)', fontSize: 13, fontWeight: 600 }}>
                  All products are above their stock thresholds.
                </div>
              )}
              {lowStock.slice(0, 4).map((it) => {
                const pct = Math.min(100, Math.round((it.available / Math.max(1, it.threshold)) * 100))
                return (
                  <div className="lsrow" key={it.productId}>
                    <Art art={artForCategory(it.category || it.title)} size={32} />
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <div style={{ fontWeight: 700, fontSize: 13 }}>{it.title}</div>
                      <div style={{ fontSize: 11.5, color: 'var(--text-faint)' }}>Threshold {it.threshold} codes</div>
                    </div>
                    <div className={'stock ' + it.level} style={{ flexDirection: 'column', alignItems: 'flex-end', gap: 4 }}>
                      <span className="num">{it.available} left</span>
                      <div className="bar" style={{ width: 60 }}>
                        <i style={{ width: pct + '%' }} />
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          <div className="acard">
            <div className="panelhead">
              <Icon name="server" size={18} />
              <h3>System health</h3>
            </div>
            <div>
              {(
                [
                  ['API server', health ? health.api : null, health ? (health.api ? 'Operational' : 'Unreachable') : 'Checking…'],
                  ['Database', health ? health.db : null, health ? (health.db ? 'Connected' : 'Unavailable') : 'Checking…'],
                ] as [string, boolean | null, string][]
              ).map(([name, ok, note]) => (
                <div className="health" key={name}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <span
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: 99,
                        background: ok === null ? 'var(--text-faint)' : ok ? 'var(--ok)' : 'var(--warn)',
                        boxShadow: ok ? '0 0 8px var(--ok)' : 'none',
                      }}
                    />
                    <span style={{ fontWeight: 700, fontSize: 13 }}>{name}</span>
                  </div>
                  <span
                    style={{
                      fontSize: 12,
                      fontWeight: 700,
                      color: ok === null ? 'var(--text-faint)' : ok ? 'var(--ok)' : 'var(--warn)',
                    }}
                  >
                    {note}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
